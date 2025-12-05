package e2e

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/net/idna"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	metav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	webhook "sigs.k8s.io/external-dns/provider/webhook/api"

	"github.com/hetzner/external-dns-hetzner-webhook/internal/provider"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
)

const (
	HetznerToken   = "HETZNER_TOKEN" // nolint:gosec // Not actually a hardcoded credentials
	HCloudDebug    = "HCLOUD_DEBUG"
	HCloudEndpoint = "HCLOUD_ENDPOINT"
)

type Cluster struct {
	*envtest.Environment
	zone *hcloud.Zone

	kubeClient     *kubernetes.Clientset
	kubeconfigPath string

	hcloudClient *hcloud.Client

	externalDNS *exec.Cmd
}

func NewCluster() (*Cluster, error) {
	ctx := context.Background()

	c := &Cluster{
		Environment: &envtest.Environment{},
	}

	hcloudClient, err := SetupHCloudClient()
	if err != nil {
		return nil, err
	}
	c.hcloudClient = hcloudClient

	result, _, err := hcloudClient.Zone.Create(
		ctx,
		hcloud.ZoneCreateOpts{
			Name: fmt.Sprintf("example-%s.com", randutil.GenerateID()),
			Mode: hcloud.ZoneModePrimary,
		},
	)
	if err != nil {
		return nil, err
	}

	if err := hcloudClient.Action.WaitFor(ctx, result.Action); err != nil {
		return nil, err
	}

	c.zone = result.Zone

	StartWebhook(hcloudClient)

	if _, err := c.Environment.Start(); err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(c.Environment.Config)
	if err != nil {
		return nil, err
	}
	c.kubeClient = client

	kubeconfigPath, err := CreateKubeConfig(c.Environment.Config)
	if err != nil {
		return nil, err
	}
	c.kubeconfigPath = kubeconfigPath

	externalDNS, err := c.StartExternalDNS(ctx)
	if err != nil {
		return nil, err
	}
	c.externalDNS = externalDNS

	return c, nil
}

func (c *Cluster) ApplyService(
	ctx context.Context,
	name string,
	annotations map[string]string,
) (*corev1.Service, error) {
	annotationsFull := make(map[string]string, len(annotations))
	for key, value := range annotations {
		fullKey := fmt.Sprintf("%s/%s", "external-dns.alpha.kubernetes.io", key)
		annotationsFull[fullKey] = value
	}

	svc := &applyconfigurationscorev1.ServiceApplyConfiguration{
		TypeMetaApplyConfiguration: metav1.TypeMetaApplyConfiguration{
			Kind:       hcloud.Ptr("Service"),
			APIVersion: hcloud.Ptr("v1"),
		},
		ObjectMetaApplyConfiguration: &metav1.ObjectMetaApplyConfiguration{
			Name:        &name,
			Annotations: annotationsFull,
		},
		Spec: &applyconfigurationscorev1.ServiceSpecApplyConfiguration{
			Type: hcloud.Ptr(corev1.ServiceTypeClusterIP),
			Ports: []applyconfigurationscorev1.ServicePortApplyConfiguration{
				{
					Port:       hcloud.Ptr[int32](80),
					TargetPort: hcloud.Ptr(intstr.FromInt32(80)),
				},
			},
		},
	}

	result, err := c.kubeClient.CoreV1().
		Services(corev1.NamespaceDefault).
		Apply(ctx, svc, v1.ApplyOptions{
			Force:        true,
			FieldManager: "external-dns-hetzner-webhook-e2e",
		})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Cluster) DeleteService(ctx context.Context, name string) error {
	err := c.kubeClient.CoreV1().
		Services(corev1.NamespaceDefault).
		Delete(ctx, name, *v1.NewDeleteOptions(60))
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Cleanup() error {
	// TestMain does not have a context we can pass through
	ctx := context.Background()

	var errs []error

	if err := c.externalDNS.Process.Signal(syscall.SIGTERM); err != nil {
		errs = append(errs, err)
	}

	result, _, err := c.hcloudClient.Zone.Delete(ctx, c.zone)
	if err != nil {
		errs = append(errs, err)
	}

	if err := c.hcloudClient.Action.WaitFor(ctx, result.Action); err != nil {
		errs = append(errs, err)
	}

	if err := c.Stop(); err != nil {
		errs = append(errs, err)
	}

	if err := os.Remove(c.kubeconfigPath); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *Cluster) WaitForRRSetCondition(
	ctx context.Context,
	subdomain string,
	condition func(zoneRRSet *hcloud.ZoneRRSet) bool,
) (*hcloud.ZoneRRSet, error) {
	backoffFn := hcloud.ExponentialBackoffWithOpts(
		hcloud.ExponentialBackoffOpts{
			Base:       time.Millisecond * 250,
			Cap:        time.Second * 60,
			Multiplier: 2.0,
		},
	)

	var retries int
	for retries < 10 {
		zoneRRSet, _, err := c.hcloudClient.Zone.GetRRSetByNameAndType(
			ctx,
			c.zone,
			subdomain,
			hcloud.ZoneRRSetTypeA,
		)
		if err != nil {
			if !hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
				return nil, err
			}
		}
		if condition(zoneRRSet) {
			return zoneRRSet, nil
		}
		retries++
		delay := backoffFn(retries)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, errors.New("reached max retries")
}

func (c *Cluster) GenerateFQDN(subdomain string) (string, error) {
	return idna.ToASCII(fmt.Sprintf("%s.%s", subdomain, c.zone.Name))
}

func (c *Cluster) StartExternalDNS(ctx context.Context) (*exec.Cmd, error) {
	cmd := exec.CommandContext( // nolint: gosec
		ctx,
		"../../external-dns",
		"--provider", "webhook",
		"--source", "service",
		"--events",
		"--interval", "60m",
		"--kubeconfig", c.kubeconfigPath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, err
}

func SetupHCloudClient() (*hcloud.Client, error) {
	hetznerToken := os.Getenv(HetznerToken)
	if hetznerToken == "" {
		return nil, fmt.Errorf("%s is empty", HetznerToken)
	}

	hcloudClientOpts := []hcloud.ClientOption{
		hcloud.WithToken(hetznerToken),
		hcloud.WithApplication("external-dns-hetzner-webhook-e2e", "dev"),
	}

	if hcloudEndpoint, ok := os.LookupEnv(HCloudEndpoint); ok {
		hcloudClientOpts = append(hcloudClientOpts, hcloud.WithEndpoint(hcloudEndpoint))
	}

	if hcloudDebugStr, ok := os.LookupEnv(HCloudDebug); ok {
		hcloudDebug, err := strconv.ParseBool(hcloudDebugStr)
		if err != nil {
			return nil, err
		}
		if hcloudDebug {
			hcloudClientOpts = append(hcloudClientOpts, hcloud.WithDebugWriter(os.Stderr))
		}
	}

	hcloudClient := hcloud.NewClient(hcloudClientOpts...)
	return hcloudClient, nil
}

func StartWebhook(client *hcloud.Client) {
	logger := slog.New(slog.DiscardHandler)
	provider := provider.NewProvider(client, logger)
	startChan := make(chan struct{})
	go webhook.StartHTTPApi(
		provider,
		startChan,
		time.Second*60,
		time.Second*60,
		"localhost:8888",
	)
	<-startChan
}

package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	webhookApi "sigs.k8s.io/external-dns/provider/webhook/api"

	"github.com/hetzner/external-dns-hetzner-webhook/internal/metrics"
	"github.com/hetzner/external-dns-hetzner-webhook/internal/provider"
	"github.com/hetzner/external-dns-hetzner-webhook/internal/version"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
)

func main() {
	logger, logLevel := newLogger()

	logger.Info("Starting webhook", "version", version.Version)

	hcloudToken, err := envutil.LookupEnvWithFile("HETZNER_TOKEN")
	if err != nil {
		logger.Error("error fetching HETZNER_TOKEN", "error", err)
		os.Exit(1)
	}

	hcloudEndpoint := os.Getenv("HCLOUD_ENDPOINT")

	clientOpts := []hcloud.ClientOption{
		hcloud.WithToken(hcloudToken),
		hcloud.WithInstrumentation(prometheus.DefaultRegisterer),
		hcloud.WithApplication("external-dns-hetzner-webhook", version.Version),
	}

	if hcloudEndpoint != "" {
		clientOpts = append(clientOpts, hcloud.WithEndpoint(hcloudEndpoint))
	}

	if logLevel == slog.LevelDebug {
		clientOpts = append(clientOpts, hcloud.WithDebugWriter(os.Stderr))
	}

	hcloudClient := hcloud.NewClient(clientOpts...)

	provider := provider.NewProvider(hcloudClient, logger)

	metricsAddr := ":8080"
	if addr, ok := os.LookupEnv("METRICS_ADDRESS"); ok {
		metricsAddr = addr
	}
	metricsServer := metrics.New(logger)
	go metricsServer.Serve(metricsAddr)

	// This webhook is recommended to run as a sidecar to external-dns with
	// localhost:8888 being their recommendation
	// https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/#provider-endpoints
	address := "localhost:8888"
	if addr, ok := os.LookupEnv("WEBHOOK_ADDRESS"); ok {
		address = addr
	}
	startChan := make(chan struct{})
	go webhookApi.StartHTTPApi(
		provider,
		startChan,
		time.Second*60,
		time.Second*60,
		address,
	)
	<-startChan
	logger.Info("Started webhook", "address", address)
	metricsServer.SetReady(true)
	metricsServer.SetHealthz(true)

	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)
	<-osSignal
	metricsServer.SetReady(false)
	metricsServer.SetHealthz(false)
}

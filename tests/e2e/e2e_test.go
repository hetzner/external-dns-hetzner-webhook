//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/idna"
	corev1 "k8s.io/api/core/v1"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
)

var cluster *Cluster

func TestMain(m *testing.M) {
	_cluster, err := NewCluster()
	if err != nil {
		fmt.Printf("%v\n", err) // nolint
		os.Exit(1)
	}
	cluster = _cluster

	code := m.Run()

	if err := cluster.Cleanup(); err != nil {
		fmt.Printf("%v\n", err) // nolint
		os.Exit(1)
	}
	os.Exit(code)
}

func TestCreateRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		recordPrefix string
		annotations  func(fqdn string) map[string]string
		expect       func(t *testing.T, service *corev1.Service, rrSet *hcloud.ZoneRRSet)
	}{
		{
			name:         "apex domain record with ttl",
			recordPrefix: "",
			annotations: func(fqdn string) map[string]string {
				return map[string]string{
					"internal-hostname": fqdn,
					"ttl":               "64",
				}
			},
			expect: func(t *testing.T, service *corev1.Service, rrSet *hcloud.ZoneRRSet) {
				assert.Equal(t, 64, *rrSet.TTL)

				require.Len(t, rrSet.Records, 1)
				assert.Equal(t, service.Spec.ClusterIP, rrSet.Records[0].Value)
			},
		},
		{
			name:         "single record with ttl",
			recordPrefix: "nginx",
			annotations: func(fqdn string) map[string]string {
				return map[string]string{
					"internal-hostname": fqdn,
					"ttl":               "64",
				}
			},
			expect: func(t *testing.T, service *corev1.Service, rrSet *hcloud.ZoneRRSet) {
				assert.Equal(t, 64, *rrSet.TTL)

				require.Len(t, rrSet.Records, 1)
				assert.Equal(t, service.Spec.ClusterIP, rrSet.Records[0].Value)
			},
		},
		{
			name:         "single record with emoji",
			recordPrefix: "nginx-🐼",
			annotations: func(fqdn string) map[string]string {
				return map[string]string{
					"internal-hostname": fqdn,
				}
			},
			expect: func(t *testing.T, service *corev1.Service, rrSet *hcloud.ZoneRRSet) {
				require.Len(t, rrSet.Records, 1)
				assert.Equal(t, service.Spec.ClusterIP, rrSet.Records[0].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			randID := randutil.GenerateID()
			recordName := fmt.Sprintf("a%s", randID) // needs to start with an alphabetic character
			if tt.recordPrefix != "" {
				recordName = fmt.Sprintf("%s-%s", tt.recordPrefix, randID)
			}

			fqdn, err := cluster.GenerateFQDN(recordName)
			require.NoError(t, err)

			recordNamePunycode, err := idna.ToASCII(recordName)
			require.NoError(t, err)

			svc, err := cluster.ApplyService(ctx, recordNamePunycode, tt.annotations(fqdn))
			require.NoError(t, err)

			zoneRRSet, err := cluster.WaitForRRSetCondition(
				ctx,
				recordNamePunycode,
				func(zoneRRSet *hcloud.ZoneRRSet) bool {
					return zoneRRSet != nil
				},
			)
			require.NoError(t, err)
			require.NotNil(t, zoneRRSet)

			tt.expect(t, svc, zoneRRSet)
		})
	}
}

func TestSimpleRecordLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	recordName := fmt.Sprintf("nginx-%s", randutil.GenerateID())
	fqdn, err := cluster.GenerateFQDN(recordName)
	require.NoError(t, err)

	t.Run("create-endpoint", func(t *testing.T) {
		_, err := cluster.ApplyService(
			ctx,
			recordName,
			map[string]string{
				"internal-hostname": fqdn,
			},
		)
		require.NoError(t, err)

		zoneRRSet, err := cluster.WaitForRRSetCondition(
			ctx,
			recordName,
			func(zoneRRSet *hcloud.ZoneRRSet) bool {
				return zoneRRSet != nil
			},
		)
		require.NoError(t, err)
		require.NotNil(t, zoneRRSet)
	})

	t.Run("delete-endpoint", func(t *testing.T) {
		err := cluster.DeleteService(ctx, recordName)
		require.NoError(t, err)

		zoneRRSet, err := cluster.WaitForRRSetCondition(
			ctx,
			recordName,
			func(zoneRRSet *hcloud.ZoneRRSet) bool {
				return zoneRRSet == nil
			},
		)
		require.NoError(t, err)
		require.Nil(t, zoneRRSet)
	})
}

func TestUpdateRecordTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	recordName := fmt.Sprintf("nginx-%s", randutil.GenerateID())
	fqdn, err := cluster.GenerateFQDN(recordName)
	require.NoError(t, err)

	t.Run("create-endpoint", func(t *testing.T) {
		_, err := cluster.ApplyService(
			ctx,
			recordName,
			map[string]string{
				"internal-hostname": fqdn,
				"ttl":               "1800",
			},
		)
		require.NoError(t, err)

		zoneRRSet, err := cluster.WaitForRRSetCondition(
			ctx,
			recordName,
			func(zoneRRSet *hcloud.ZoneRRSet) bool {
				return zoneRRSet != nil
			},
		)
		require.NoError(t, err)
		require.NotNil(t, zoneRRSet)
		assert.Equal(t, 1800, *zoneRRSet.TTL)
	})

	t.Run("update-endpoint", func(t *testing.T) {
		_, err := cluster.ApplyService(
			ctx,
			recordName,
			map[string]string{
				"internal-hostname": fqdn,
				"ttl":               "3600",
			},
		)
		require.NoError(t, err)

		zoneRRSet, err := cluster.WaitForRRSetCondition(
			ctx,
			recordName,
			func(zoneRRSet *hcloud.ZoneRRSet) bool {
				return zoneRRSet != nil && *zoneRRSet.TTL != 1800
			},
		)
		require.NoError(t, err)
		require.NotNil(t, zoneRRSet)
		assert.Equal(t, 3600, *zoneRRSet.TTL)
	})
}

package provider

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestFindZoneByHostname(t *testing.T) {
	tests := []struct {
		name     string
		zones    []*hcloud.Zone
		hostname string
		want     *hcloud.Zone
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name:     "no zone",
			zones:    nil,
			hostname: "",
			want:     nil,
			wantErr:  assert.Error,
		},
		{
			name: "one zone",
			zones: []*hcloud.Zone{
				{
					Name: "example.com",
				},
			},
			hostname: "testing.example.com",
			want: &hcloud.Zone{
				Name: "example.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple zones",
			zones: []*hcloud.Zone{
				{
					Name: "example.com",
				},
				{
					Name: "example.org",
				},
				{
					Name: "example.gov",
				},
			},
			hostname: "testing.example.org",
			want: &hcloud.Zone{
				Name: "example.org",
			},
			wantErr: assert.NoError,
		},
		{
			name: "no shorter suffix match",
			zones: []*hcloud.Zone{
				{
					Name: "example.org",
				},
				{
					Name: "prefix-example.org",
				},
			},
			hostname: "testing.prefix-example.org",
			want: &hcloud.Zone{
				Name: "prefix-example.org",
			},
			wantErr: assert.NoError,
		},
		{
			name: "no longer suffix match",
			zones: []*hcloud.Zone{
				{
					Name: "example.org",
				},
				{
					Name: "prefix-example.org",
				},
			},
			hostname: "testing.example.org",
			want: &hcloud.Zone{
				Name: "example.org",
			},
			wantErr: assert.NoError,
		},
		{
			name: "no zone found",
			zones: []*hcloud.Zone{
				{
					Name: "example.com",
				},
				{
					Name: "example.org",
				},
				{
					Name: "example.gov",
				},
			},
			hostname: "testing.example.de",
			want:     nil,
			wantErr:  assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findZoneByHostname(tt.zones, tt.hostname)
			if !tt.wantErr(t, err, fmt.Sprintf("findZoneByHostname(%v, %v)", tt.zones, tt.hostname)) {
				return
			}
			assert.Equalf(t, tt.want, got, "findZoneByHostname(%v, %v)", tt.zones, tt.hostname)
		})
	}
}

func TestFindZoneByHostnameDomainApex(t *testing.T) {
	tests := []struct {
		name     string
		zones    []*hcloud.Zone
		hostname string
		want     *hcloud.Zone
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "domain apex should match zone exactly",
			zones: []*hcloud.Zone{
				{
					Name: "example.com",
				},
			},
			hostname: "example.com",
			want: &hcloud.Zone{
				Name: "example.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "subdomain should match zone",
			zones: []*hcloud.Zone{
				{
					Name: "example.com",
				},
			},
			hostname: "sub.example.com",
			want: &hcloud.Zone{
				Name: "example.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "domain apex should not match longer zone name",
			zones: []*hcloud.Zone{
				{
					Name: "sub.example.com",
				},
			},
			hostname: "example.com",
			want:     nil,
			wantErr:  assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findZoneByHostname(tt.zones, tt.hostname)
			if !tt.wantErr(t, err, fmt.Sprintf("findZoneByHostname(%v, %v)", tt.zones, tt.hostname)) {
				return
			}
			assert.Equalf(t, tt.want, got, "findZoneByHostname(%v, %v)", tt.zones, tt.hostname)
		})
	}
}

func TestParseArrayFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{
			name: "empty",
			env:  "",
			want: nil,
		},
		{
			name: "single",
			env:  "example.org",
			want: []string{"example.org"},
		},
		{
			name: "multiple",
			env:  "example.org,example.gov",
			want: []string{"example.org", "example.gov"},
		},
		{
			name: "multiple with whitespaces",
			env:  "example.org,	example.gov,  example.com",
			want: []string{"example.org", "example.gov", "example.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envKey := "TEST_DOMAINS"
			t.Setenv(envKey, tt.env)
			assert.Equalf(t, tt.want, parseArrayFromEnv(envKey), "parseArrayFromEnv(%v)", tt.env)
		})
	}
}

func TestAdjustCNAMETarget(t *testing.T) {
	tests := []struct {
		name    string
		zone    *hcloud.Zone
		dnsName string
		target  string
		// Expected target with ALLOW_RELATIVE_CNAME_TARGETS disabled (default), every
		// target is made absolute.
		wantTarget string
		// Expected target with ALLOW_RELATIVE_CNAME_TARGETS enabled, targets which do not
		// end with a public suffix stay relative.
		wantTargetRelativeAllowed string
	}{
		{
			name:                      "absolute target in same zone",
			zone:                      &hcloud.Zone{Name: "example.de"},
			dnsName:                   "www.example.de",
			target:                    "node1.example.de",
			wantTarget:                "node1.example.de.",
			wantTargetRelativeAllowed: "node1.example.de.",
		},
		{
			name:                      "absolute target in other zone",
			zone:                      &hcloud.Zone{Name: "example.de"},
			dnsName:                   "www.example.de",
			target:                    "node1.other.de",
			wantTarget:                "node1.other.de.",
			wantTargetRelativeAllowed: "node1.other.de.",
		},
		{
			name:                      "relative target in same zone",
			zone:                      &hcloud.Zone{Name: "example.de"},
			dnsName:                   "www.example.de",
			target:                    "node1",
			wantTarget:                "node1.",
			wantTargetRelativeAllowed: "node1",
		},
		{
			// external-dns always strips the trailing dots, but we want to make sure we
			// handle this case if it were to change.
			name:                      "absolute target in same zone with trailing dot",
			zone:                      &hcloud.Zone{Name: "example.de"},
			dnsName:                   "www.example.de",
			target:                    "node1.example.de.",
			wantTarget:                "node1.example.de.",
			wantTargetRelativeAllowed: "node1.example.de.",
		},
		{
			// external-dns always strips the trailing dots, but we want to make sure we
			// handle this case if it were to change.
			name:                      "absolute target in other zone with trailing dot",
			zone:                      &hcloud.Zone{Name: "example.de"},
			dnsName:                   "www.example.de",
			target:                    "node1.other.de.",
			wantTarget:                "node1.other.de.",
			wantTargetRelativeAllowed: "node1.other.de.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("relative targets disallowed", func(t *testing.T) {
				target := adjustCNAMETarget(tt.target, false)
				require.Equal(t, tt.wantTarget, target)
			})

			t.Run("relative targets allowed", func(t *testing.T) {
				target := adjustCNAMETarget(tt.target, true)
				require.Equal(t, tt.wantTargetRelativeAllowed, target)
			})
		})
	}
}

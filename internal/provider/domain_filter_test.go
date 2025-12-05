package provider

import (
	"io"
	"log/slog"
	"reflect"
	"regexp"
	"testing"

	"sigs.k8s.io/external-dns/endpoint"
)

func TestGetDomainFilter(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want func() endpoint.DomainFilterInterface
	}{
		{
			name: "include domain filter",
			envs: map[string]string{
				"INCLUDE_DOMAIN_FILTER": "allow.com,allow2.com,allow3.com",
			},
			want: func() endpoint.DomainFilterInterface {
				return endpoint.NewDomainFilter(
					[]string{"allow.com", "allow2.com", "allow3.com"},
				)
			},
		},
		{
			name: "include and exclude domain filter",
			envs: map[string]string{
				"INCLUDE_DOMAIN_FILTER": "allow.com",
				"EXCLUDE_DOMAIN_FILTER": "deny.com,deny2.com",
			},
			want: func() endpoint.DomainFilterInterface {
				return endpoint.NewDomainFilterWithExclusions(
					[]string{"allow.com"},
					[]string{"deny.com", "deny2.com"},
				)
			},
		},
		{
			name: "include regexp domain filter",
			envs: map[string]string{
				"INCLUDE_REGEXP_DOMAIN_FILTER": "allow-[0-9a-z]*\\.com",
			},
			want: func() endpoint.DomainFilterInterface {
				return endpoint.NewRegexDomainFilter(
					regexp.MustCompile(`allow-[0-9a-z]*\.com`),
					nil,
				)
			},
		},
		{
			name: "include and exclude regexp domain filter",
			envs: map[string]string{
				"INCLUDE_REGEXP_DOMAIN_FILTER": "allow-[0-9a-z]*\\.com",
				"EXCLUDE_REGEXP_DOMAIN_FILTER": "deny-[0-9a-z]*\\.com",
			},
			want: func() endpoint.DomainFilterInterface {
				return endpoint.NewRegexDomainFilter(
					regexp.MustCompile(`allow-[0-9a-z]*\.com`),
					regexp.MustCompile(`deny-[0-9a-z]*\.com`),
				)
			},
		},
		{
			name: "every config option set",
			envs: map[string]string{
				"INCLUDE_DOMAIN_FILTER":        "allow.com",
				"EXCLUDE_DOMAIN_FILTER":        "deny.com,deny2.com",
				"INCLUDE_REGEXP_DOMAIN_FILTER": "allow-[0-9a-z]*\\.com",
				"EXCLUDE_REGEXP_DOMAIN_FILTER": "deny-[0-9a-z]*\\.com",
			},
			want: func() endpoint.DomainFilterInterface {
				return endpoint.NewDomainFilterWithExclusions(
					[]string{"allow.com"},
					[]string{"deny.com", "deny2.com"},
				)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

			for k, v := range tt.envs {
				t.Setenv(k, v)
			}

			want := tt.want()
			if got := getDomainFilter(logger); !reflect.DeepEqual(got, want) {
				t.Errorf("getDomainFilter() = %v, want %v", got, want)
			}
		})
	}
}

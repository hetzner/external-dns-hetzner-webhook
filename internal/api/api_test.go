package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/external-dns/endpoint"

	"github.com/hetzner/external-dns-hetzner-webhook/internal/testutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func TestAdjustEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		incoming []*endpoint.Endpoint
		expected []*endpoint.Endpoint
	}{
		{
			name: "convert to lowercase",
			incoming: []*endpoint.Endpoint{
				{
					DNSName: "MyDomain.example.com",
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName: "mydomain.example.com",
				},
			},
		},
		{
			name: "convert to punicode",
			incoming: []*endpoint.Endpoint{
				{
					DNSName: "mydømain.example.com",
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName: "xn--mydmain-s1a.example.com",
				},
			},
		},
		{
			name: "trim trailing dot",
			incoming: []*endpoint.Endpoint{
				{
					DNSName: "mydomain.example.com.",
				},
			},
			expected: []*endpoint.Endpoint{
				{
					DNSName: "mydomain.example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, client, logger := testutil.MakeTestUtils(t)

			hetznerProvider := NewProvider(client, logger)

			actual, err := hetznerProvider.AdjustEndpoints(tt.incoming)
			require.NoError(t, err)
			assert.Len(t, actual, len(tt.expected))

			for i, ep := range actual {
				assert.Equal(t, tt.expected[i].DNSName, ep.DNSName)
			}
		})
	}
}

func TestRecords(t *testing.T) {
	tests := []struct {
		name       string
		zoneNameFn func(id string) string
		expectedFn func(zoneName string) []*endpoint.Endpoint
		mocksFn    func(zoneName string) []mockutil.Request
	}{
		{
			name:       "single zone single record",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			expectedFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName: fmt.Sprintf("mydomain.%s", zoneName),
						Targets: endpoint.NewTargets("127.0.0.1"),
					}}
			},
			mocksFn: func(zoneName string) []mockutil.Request {
				return []mockutil.Request{
					testutil.GetZonesMock(zoneName),
					testutil.GetRRSetsMock(
						"mydomain",
						string(hcloud.ZoneRRSetTypeA),
						[]schema.ZoneRRSetRecord{
							{
								Value: "127.0.0.1",
							},
						},
					),
				}
			},
		},
		{
			name:       "single zone domain apex record",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			expectedFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName: zoneName, // Domain apex
						Targets: endpoint.NewTargets("127.0.0.1"),
					}}
			},
			mocksFn: func(zoneName string) []mockutil.Request {
				return []mockutil.Request{
					testutil.GetZonesMock(zoneName),
					testutil.GetRRSetsMock(
						"@", // Domain apex RRSet name
						string(hcloud.ZoneRRSetTypeA),
						[]schema.ZoneRRSetRecord{
							{
								Value: "127.0.0.1",
							},
						},
					),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, ctx, server, client, logger := testutil.MakeTestUtils(t)

			zoneName := tt.zoneNameFn(runID)
			expected := tt.expectedFn(zoneName)

			server.Expect(tt.mocksFn(zoneName))

			hetznerProvider := NewProvider(client, logger)

			actual, err := hetznerProvider.Records(ctx)
			require.NoError(t, err)
			assert.Len(t, actual, len(expected))

			for i, ep := range actual {
				assert.Equal(t, expected[i].DNSName, ep.DNSName)
				assert.Len(t, expected[i].Targets, ep.Targets.Len())
				for j, target := range ep.Targets {
					assert.Equal(t, expected[i].Targets[j], target)
				}
			}
		})
	}
}

func TestApplyCreateChanges(t *testing.T) {
	tests := []struct {
		name             string
		zoneNameFn       func(id string) string
		inputEndpointsFn func(zoneName string) []*endpoint.Endpoint
		mocksFn          func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request
	}{
		{
			name:       "create single rrset",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			inputEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
					},
				}
			},
			mocksFn: func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range inputEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "POST",
						Path:   fmt.Sprintf("/zones/%s/rrsets/%s/%s/actions/add_records", zoneName, "test", ep.RecordType),
						Status: 200,
						Want: func(t *testing.T, r *http.Request) {
							request := schema.ZoneRRSetAddRecordsRequest{}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
							assert.Equal(
								t,
								schema.ZoneRRSetAddRecordsRequest{
									Records: []schema.ZoneRRSetRecord{
										{Value: "127.0.0.1"},
									},
								},
								request,
							)
						},
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "add_rrset_records", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
		{
			name:       "create domain apex rrset",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			inputEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    zoneName, // Domain apex
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
					},
				}
			},
			mocksFn: func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range inputEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "POST",
						Path:   fmt.Sprintf("/zones/%s/rrsets/@/%s/actions/add_records", zoneName, ep.RecordType),
						Status: 200,
						Want: func(t *testing.T, r *http.Request) {
							request := schema.ZoneRRSetAddRecordsRequest{}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
							assert.Equal(
								t,
								schema.ZoneRRSetAddRecordsRequest{
									Records: []schema.ZoneRRSetRecord{
										{Value: "127.0.0.1"},
									},
								},
								request,
							)
						},
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "add_rrset_records", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, ctx, server, client, logger := testutil.MakeTestUtils(t)

			zoneName := tt.zoneNameFn(runID)
			inputEndpoints := tt.inputEndpointsFn(zoneName)
			server.Expect(tt.mocksFn(zoneName, inputEndpoints))

			hetznerProvider := NewProvider(client, logger)

			zones := []*hcloud.Zone{{Name: zoneName}}

			err := hetznerProvider.applyCreateChanges(ctx, zones, inputEndpoints)
			require.NoError(t, err)
		})
	}
}

func TestApplyDeleteChanges(t *testing.T) {
	tests := []struct {
		name             string
		zoneNameFn       func(id string) string
		inputEndpointsFn func(zoneName string) []*endpoint.Endpoint
		mocksFn          func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request
	}{
		{
			name:       "delete single rrset",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			inputEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
					},
				}
			},
			mocksFn: func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range inputEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "DELETE",
						Path:   fmt.Sprintf("/zones/%s/rrsets/%s/%s", zoneName, "test", ep.RecordType),
						Status: 200,
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "delete_rrset", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
		{
			name:       "delete domain apex rrset",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			inputEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    zoneName, // Domain apex
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
					},
				}
			},
			mocksFn: func(zoneName string, inputEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range inputEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "DELETE",
						Path:   fmt.Sprintf("/zones/%s/rrsets/@/%s", zoneName, ep.RecordType),
						Status: 200,
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "delete_rrset", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, ctx, server, client, logger := testutil.MakeTestUtils(t)

			zoneName := tt.zoneNameFn(runID)
			inputEndpoints := tt.inputEndpointsFn(zoneName)
			server.Expect(tt.mocksFn(zoneName, inputEndpoints))

			hetznerProvider := NewProvider(client, logger)

			zones := []*hcloud.Zone{{Name: zoneName}}

			err := hetznerProvider.applyDeleteChanges(ctx, zones, inputEndpoints)
			require.NoError(t, err)
		})
	}
}

func TestApplyUpdateChanges(t *testing.T) {
	tests := []struct {
		name           string
		zoneNameFn     func(id string) string
		oldEndpointsFn func(zoneName string) []*endpoint.Endpoint
		newEndpointsFn func(zoneName string) []*endpoint.Endpoint
		mocksFn        func(zoneName string, oldEndpoints []*endpoint.Endpoint, newEndpoints []*endpoint.Endpoint) []mockutil.Request
	}{
		{
			name:       "update ttl",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			oldEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
						RecordTTL:  endpoint.TTL(3600),
					},
				}
			},
			newEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
						RecordTTL:  endpoint.TTL(1800),
					},
				}
			},
			mocksFn: func(zoneName string, oldEndpoints []*endpoint.Endpoint, newEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range oldEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "POST",
						Path:   fmt.Sprintf("/zones/%s/rrsets/%s/%s/actions/change_ttl", zoneName, "test", ep.RecordType),
						Status: 200,
						Want: func(t *testing.T, r *http.Request) {
							request := schema.ZoneChangeTTLRequest{}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
							assert.Equal(t, schema.ZoneChangeTTLRequest{TTL: 1800}, request)
						},
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "change_rrset_ttl", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
		{
			name:       "update records",
			zoneNameFn: func(id string) string { return fmt.Sprintf("example-%s.com", id) },
			oldEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"127.0.0.1"},
					},
				}
			},
			newEndpointsFn: func(zoneName string) []*endpoint.Endpoint {
				return []*endpoint.Endpoint{
					{
						DNSName:    fmt.Sprintf("%s.%s", "test", zoneName),
						RecordType: "A",
						Targets:    []string{"192.168.0.1"},
					},
				}
			},
			mocksFn: func(zoneName string, oldEndpoints []*endpoint.Endpoint, newEndpoints []*endpoint.Endpoint) []mockutil.Request {
				mocks := make([]mockutil.Request, 0)
				for _, ep := range oldEndpoints {
					mocks = append(mocks, mockutil.Request{
						Method: "POST",
						Path:   fmt.Sprintf("/zones/%s/rrsets/%s/%s/actions/set_records", zoneName, "test", ep.RecordType),
						Status: 200,
						Want: func(t *testing.T, r *http.Request) {
							request := schema.ZoneRRSetSetRecordsRequest{}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
							assert.Equal(
								t,
								schema.ZoneRRSetSetRecordsRequest{
									Records: []schema.ZoneRRSetRecord{{Value: "192.168.0.1"}},
								},
								request,
							)
						},
						JSON: schema.ActionGetResponse{
							Action: schema.Action{ID: 1, Command: "set_rrset_records", Status: "success"},
						},
					})
				}
				return mocks
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, ctx, server, client, logger := testutil.MakeTestUtils(t)

			zoneName := tt.zoneNameFn(runID)
			oldEndpoints := tt.oldEndpointsFn(zoneName)
			newEndpoints := tt.newEndpointsFn(zoneName)
			server.Expect(tt.mocksFn(zoneName, oldEndpoints, newEndpoints))

			hetznerProvider := NewProvider(client, logger)

			zones := []*hcloud.Zone{{Name: zoneName}}

			err := hetznerProvider.applyUpdateChanges(ctx, zones, oldEndpoints, newEndpoints)
			assert.NoError(t, err)
		})
	}
}

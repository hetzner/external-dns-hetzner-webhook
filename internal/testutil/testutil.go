package testutil

import (
	"context"
	"log/slog"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

func MakeTestUtils(t *testing.T) (
	string,
	context.Context,
	*mockutil.Server,
	*hcloud.Client,
	*slog.Logger,
) {
	t.Helper()

	runID := randutil.GenerateID()

	ctx := t.Context()

	server := mockutil.NewServer(t, nil)

	client := hcloud.NewClient(
		hcloud.WithEndpoint(server.URL),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0), MaxRetries: 5}),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)

	logger := slog.New(slog.DiscardHandler)

	return runID, ctx, server, client, logger
}

func GetZonesMock(zoneName string) mockutil.Request {
	return mockutil.Request{
		Method: "GET", Path: "/zones?page=1&per_page=50",
		Status: 200,
		JSON: schema.ZoneListResponse{
			Zones: []schema.Zone{
				{
					ID:   1,
					Name: zoneName,
				},
			},
		},
	}
}

func GetRRSetsMock(
	rrSetName string,
	rrSetType string,
	rrSetRecords []schema.ZoneRRSetRecord,
) mockutil.Request {
	return mockutil.Request{
		Method: "GET", Path: "/zones/1/rrsets?page=1&per_page=100",
		Status: 200,
		JSON: schema.ZoneRRSetListResponse{
			RRSets: []schema.ZoneRRSet{
				{
					Name:    rrSetName,
					Type:    rrSetType,
					Records: rrSetRecords,
				},
			},
		},
	}
}

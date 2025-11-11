package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/net/idna"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type Provider struct {
	client *hcloud.Client
	logger *slog.Logger
}

func NewProvider(
	client *hcloud.Client,
	logger *slog.Logger,
) *Provider {
	return &Provider{
		client: client,
		logger: logger,
	}
}

func (p *Provider) GetDomainFilter() endpoint.DomainFilterInterface {
	return getDomainFilter(p.logger)
}

func (p *Provider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	for _, ep := range endpoints {
		ep.DNSName = strings.ToLower(ep.DNSName)
		ep.DNSName = strings.TrimSuffix(ep.DNSName, ".")
		dnsName, err := idna.ToASCII(ep.DNSName)
		if err != nil {
			return nil, err
		}

		p.logger.Debug(
			"adjusted endpoint",
			"input", ep.DNSName,
			"output", dnsName,
		)
		ep.DNSName = dnsName
	}
	return endpoints, nil
}

func (p *Provider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.client.Zone.All(ctx)
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint
	for _, zone := range zones {
		rrsets, err := p.client.Zone.AllRRSetsWithOpts(ctx, zone, hcloud.ZoneRRSetListOpts{
			ListOpts: hcloud.ListOpts{
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, rrset := range rrsets {
			var dnsName string
			if rrset.Name == "@" {
				dnsName = zone.Name
			} else {
				dnsName = fmt.Sprintf("%s.%s", rrset.Name, zone.Name)
			}
			ep := &endpoint.Endpoint{
				DNSName:    dnsName,
				RecordType: string(rrset.Type),
			}

			ep.Targets = make([]string, 0, len(rrset.Records))
			for _, record := range rrset.Records {
				ep.Targets = append(ep.Targets, record.Value)
			}

			if rrset.TTL != nil {
				ep.RecordTTL = endpoint.TTL(*rrset.TTL)
			}

			endpoints = append(endpoints, ep)
		}
	}

	p.logger.Info(
		"fetched records for all available zones",
		"amount", len(endpoints),
	)

	return endpoints, nil
}

func (p *Provider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	zones, err := p.client.Zone.All(ctx)
	if err != nil {
		return err
	}

	if err := p.applyDeleteChanges(ctx, zones, changes.Delete); err != nil {
		return err
	}

	if err := p.applyUpdateChanges(ctx, zones, changes.UpdateOld, changes.UpdateNew); err != nil {
		return err
	}

	if err := p.applyCreateChanges(ctx, zones, changes.Create); err != nil {
		return err
	}

	return nil
}

func (p *Provider) applyCreateChanges(
	ctx context.Context,
	zones []*hcloud.Zone,
	endpoints []*endpoint.Endpoint,
) error {
	actions := make([]*hcloud.Action, 0, len(endpoints))

	for _, ep := range endpoints {
		zone, err := findZoneByHostname(zones, ep.DNSName)
		if err != nil {
			return err
		}

		zoneRRSetName := getZoneRRSetName(ep.DNSName, zone)

		zoneRRSet := &hcloud.ZoneRRSet{
			Zone: zone,
			Name: zoneRRSetName,
			Type: hcloud.ZoneRRSetType(ep.RecordType),
		}

		var records []hcloud.ZoneRRSetRecord
		for _, target := range ep.Targets {
			records = append(records, hcloud.ZoneRRSetRecord{Value: target})
		}

		opts := hcloud.ZoneRRSetAddRecordsOpts{
			Records: records,
		}
		if ep.RecordTTL.IsConfigured() {
			opts.TTL = hcloud.Ptr(int(ep.RecordTTL))
		}

		action, _, err := p.client.Zone.AddRRSetRecords(ctx, zoneRRSet, opts)
		if err != nil {
			return err
		}
		actions = append(actions, action)

		p.logger.Info(
			"requested endpoint creation",
			"type", ep.RecordType,
			"name", zoneRRSetName,
		)
	}

	if err := p.waitActions(ctx, actions...); err != nil {
		return err
	}

	if len(actions) > 0 {
		p.logger.Info("endpoints creation completed")
	}

	return nil
}

func (p *Provider) applyDeleteChanges(
	ctx context.Context,
	zones []*hcloud.Zone,
	endpoints []*endpoint.Endpoint,
) error {
	actions := make([]*hcloud.Action, 0, len(endpoints))

	for _, ep := range endpoints {
		zone, err := findZoneByHostname(zones, ep.DNSName)
		if err != nil {
			return err
		}

		zoneRRSetName := getZoneRRSetName(ep.DNSName, zone)

		zoneRRSet := &hcloud.ZoneRRSet{
			Zone: zone,
			Name: zoneRRSetName,
			Type: hcloud.ZoneRRSetType(ep.RecordType),
		}

		result, _, err := p.client.Zone.DeleteRRSet(ctx, zoneRRSet)
		if err != nil {
			return err
		}
		actions = append(actions, result.Action)

		p.logger.Info(
			"requested endpoint deletion",
			"type", ep.RecordType,
			"name", zoneRRSetName,
		)
	}

	if err := p.waitActions(ctx, actions...); err != nil {
		return err
	}

	if len(actions) > 0 {
		p.logger.Info("endpoints deletion completed")
	}

	return nil
}

func (p *Provider) applyUpdateChanges(
	ctx context.Context,
	zones []*hcloud.Zone,
	endpointsOld []*endpoint.Endpoint,
	endpointsNew []*endpoint.Endpoint,
) error {
	actions := make([]*hcloud.Action, 0, 2*len(endpointsOld))

	for i := range endpointsNew {
		zone, err := findZoneByHostname(zones, endpointsOld[i].DNSName)
		if err != nil {
			return err
		}

		zoneRRSetName := getZoneRRSetName(endpointsOld[i].DNSName, zone)

		// Update TTL
		ttlOld := int(endpointsOld[i].RecordTTL)
		ttlNew := int(endpointsNew[i].RecordTTL)
		if ttlOld != ttlNew {
			zoneRRSet := &hcloud.ZoneRRSet{
				Zone: zone,
				Name: zoneRRSetName,
				Type: hcloud.ZoneRRSetType(endpointsOld[i].RecordType),
			}

			action, _, err := p.client.Zone.ChangeRRSetTTL(
				ctx,
				zoneRRSet,
				hcloud.ZoneRRSetChangeTTLOpts{
					TTL: &ttlNew,
				},
			)
			if err != nil {
				return err
			}
			actions = append(actions, action)

			p.logger.Info(
				"requested endpoint ttl update",
				"type", endpointsOld[i].RecordType,
				"name", zoneRRSetName,
				"old", endpointsOld[i].RecordTTL,
				"new", endpointsNew[i].RecordTTL,
			)
		}

		// Update Records
		if !endpointsOld[i].Targets.Same(endpointsNew[i].Targets) {
			zoneRRSet := &hcloud.ZoneRRSet{
				Zone: zone,
				Name: zoneRRSetName,
				Type: hcloud.ZoneRRSetType(endpointsOld[i].RecordType),
			}

			records := make([]hcloud.ZoneRRSetRecord, 0, len(endpointsNew[i].Targets))
			for _, target := range endpointsNew[i].Targets {
				records = append(records, hcloud.ZoneRRSetRecord{Value: target})
			}

			action, _, err := p.client.Zone.SetRRSetRecords(
				ctx,
				zoneRRSet,
				hcloud.ZoneRRSetSetRecordsOpts{
					Records: records,
				},
			)
			if err != nil {
				return err
			}
			actions = append(actions, action)

			p.logger.Info(
				"requested endpoint records update",
				"type", endpointsOld[i].RecordType,
				"name", endpointsOld[i].DNSName,
				"old", endpointsOld[i].Targets,
				"new", endpointsNew[i].Targets,
			)
		}
	}

	if err := p.waitActions(ctx, actions...); err != nil {
		return err
	}

	if len(actions) > 0 {
		p.logger.Info("endpoints update completed")
	}

	return nil
}

func (p *Provider) waitActions(ctx context.Context, actions ...*hcloud.Action) error {
	failed := false
	if err := p.client.Action.WaitForFunc(ctx, func(update *hcloud.Action) error {
		if err := update.Error(); err != nil {
			p.logger.Error("action failed", "err", err)
			failed = true
		}
		return nil
	}, actions...); err != nil {
		return err
	}

	if failed {
		return errors.New("waiting for actions failed")
	}
	return nil
}

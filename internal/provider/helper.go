package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func findZoneByHostname(zones []*hcloud.Zone, hostname string) (*hcloud.Zone, error) {
	var match *hcloud.Zone
	for _, zone := range zones {
		if strings.HasSuffix(hostname, zone.Name) {
			// Ensures the longest match is returned
			if match == nil || len(match.Name) < len(zone.Name) {
				match = zone
			}
		}
	}
	if match == nil {
		return nil, fmt.Errorf("could not find zone with hostname: %s", hostname)
	}
	return match, nil
}

func getZoneRRSetName(dnsName string, zone *hcloud.Zone) string {
	zoneRRSetName := strings.TrimSuffix(dnsName, fmt.Sprintf(".%s", zone.Name))
	// For domain apex records, use "@" as the RRSet name
	if zoneRRSetName == zone.Name {
		zoneRRSetName = "@"
	}
	return zoneRRSetName
}

func parseArrayFromEnv(env string) []string {
	envVal := os.Getenv(env)
	if envVal == "" {
		return nil
	}
	parts := strings.Split(envVal, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

func adjustCNAMETarget(zones []*hcloud.Zone, dnsName string, target string) (string, error) {
	zone, err := findZoneByHostname(zones, dnsName)
	if err != nil {
		return "", err
	}

	var adjustedTarget string
	zoneSuffix := fmt.Sprintf(".%s", zone.Name)
	absoluteZoneSuffix := fmt.Sprintf(".%s.", zone.Name)
	switch {
	case strings.HasSuffix(target, zoneSuffix):
		adjustedTarget = strings.TrimSuffix(target, zoneSuffix)
	case strings.HasSuffix(target, absoluteZoneSuffix):
		adjustedTarget = strings.TrimSuffix(target, absoluteZoneSuffix)
	default:
		// ensure CNAME targets in different zones end in a dot
		adjustedTarget = strings.TrimSuffix(target, ".") + "."
	}

	return adjustedTarget, nil
}

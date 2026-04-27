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

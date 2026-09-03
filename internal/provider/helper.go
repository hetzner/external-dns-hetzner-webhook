package provider

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"

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

// adjustCNAMETarget updates the CNAME record value.
//
// The target is converted to its IDNA (Punycode) form, because the Hetzner DNS API rejects
// non-ASCII characters in record values.
//
// Additionally the trailing dot is restored, to work around the removal of trailing dots in
// external-dns (https://github.com/kubernetes-sigs/external-dns/issues/6145).
//
// By default every target is made absolute by appending a trailing dot, this means relative targets cannot be
// used.
//
// With allowRelativeCNAMETargets enabled, a target is only made absolute if it ends with a public suffix
// (https://publicsuffix.org/), every other target is left untouched and stays relative to the zone of the
// record. A relative target which ends with a public suffix is not supported in this mode, because it cannot
// be told apart from an absolute target, instead users must use an absolute target.
//
// For example given the zone "example.com", the relative target "embedded.other.de" (without trailing dot) must
// use the following absolute CNAME target "embedded.other.de.example.com."
//
//	external-dns.kubernetes.io/hostname: "www.example.com"
//	external-dns.kubernetes.io/target: "embedded.other.de.example.com."
func adjustCNAMETarget(target string, allowRelativeCNAMETargets bool) (string, error) {
	asciiTarget, err := idna.ToASCII(target)
	if err != nil {
		return "", fmt.Errorf("could not convert CNAME target to ASCII: %s: %w", target, err)
	}

	if !allowRelativeCNAMETargets {
		return strings.TrimSuffix(asciiTarget, ".") + ".", nil
	}

	suffix, icann := publicsuffix.PublicSuffix(asciiTarget)
	// If target is ICANN or privately managed, append a trailing dot
	// https://pkg.go.dev/golang.org/x/net/publicsuffix#example-PublicSuffix-Manager
	if icann || strings.IndexByte(suffix, '.') >= 0 {
		return strings.TrimSuffix(asciiTarget, ".") + ".", nil
	}

	return asciiTarget, nil
}

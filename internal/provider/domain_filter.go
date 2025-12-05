package provider

import (
	"log/slog"
	"os"
	"regexp"

	"sigs.k8s.io/external-dns/endpoint"
)

const (
	IncludeDomainFilter       = "INCLUDE_DOMAIN_FILTER"
	ExcludeDomainFilter       = "EXCLUDE_DOMAIN_FILTER"
	IncludeRegexpDomainFilter = "INCLUDE_REGEXP_DOMAIN_FILTER"
	ExcludeRegexpDomainFilter = "EXCLUDE_REGEXP_DOMAIN_FILTER"
)

func getDomainFilter(logger *slog.Logger) endpoint.DomainFilterInterface {
	includeDomainFilter := parseArrayFromEnv(IncludeDomainFilter)
	excludeDomainFilter := parseArrayFromEnv(ExcludeDomainFilter)
	includeRegexpDomainFilter := os.Getenv(IncludeRegexpDomainFilter)
	excludeRegexpDomainFilter := os.Getenv(ExcludeRegexpDomainFilter)

	if includeDomainFilter != nil {
		if excludeDomainFilter != nil {
			logger.Info("using domain filter",
				"include-domain-filter", includeDomainFilter,
				"exclude-domain-filter", excludeDomainFilter,
			)
			return endpoint.NewDomainFilterWithExclusions(
				includeDomainFilter,
				excludeDomainFilter,
			)
		}
		logger.Info("using domain filter",
			"include-domain-filter", includeDomainFilter,
		)
		return endpoint.NewDomainFilter(includeDomainFilter)
	}

	var includeRegexp *regexp.Regexp
	if includeRegexpDomainFilter != "" {
		includeRegexp = regexp.MustCompile(includeRegexpDomainFilter)
	}

	var excludeRegexp *regexp.Regexp
	if excludeRegexpDomainFilter != "" {
		excludeRegexp = regexp.MustCompile(excludeRegexpDomainFilter)
	}

	if includeRegexp != nil || excludeRegexp != nil {
		logger.Info("using regexp domain filter",
			"include-regexp-domain-filter", includeRegexpDomainFilter,
			"exclude-regexp-domain-filter", excludeRegexpDomainFilter,
		)
		return endpoint.NewRegexDomainFilter(includeRegexp, excludeRegexp)
	}

	logger.Info("no domain filter in use")
	return &endpoint.DomainFilter{}
}

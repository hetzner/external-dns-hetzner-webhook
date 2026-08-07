# Configuration reference

This page references the different configurations for the External DNS Hetzner webhook.

## Supported environment variables

| Env                            | Type                  | Default          | Description                                                                                                                              |
| ------------------------------ | --------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `HETZNER_TOKEN`                | string (**required**) |                  | Hetzner Cloud API token.                                                                                                                 |
| `INCLUDE_DOMAIN_FILTER`        | list of string        |                  | Inclusive filtering of target zones via domain suffixes. See [domain filters](#domain-filters) for more details.                         |
| `EXCLUDE_DOMAIN_FILTER`        | list of string        |                  | Exclusive filtering of target zones via domain suffixes. See [domain filters](#domain-filters) for more details.                         |
| `INCLUDE_REGEXP_DOMAIN_FILTER` | string                |                  | Inclusive filtering of target zones via regular expressions. See [domain filters](#domain-filters) for more details.                     |
| `EXCLUDE_REGEXP_DOMAIN_FILTER` | string                |                  | Exclusive filtering of target zones via regular expressions. See [domain filters](#domain-filters) for more details.                     |
| `ALLOW_RELATIVE_CNAME_TARGETS` | boolean               | `false`          | Keep CNAME targets relative to the zone. See [allow relative CNAME targets](../guides/allow-relative-cname-targets.md) for more details. |
| `LOG_LEVEL`                    | string                | `info`           | Log level of the webhook. Accepted values are `debug`, `info`, `warn`, `error`.                                                          |
| `WEBHOOK_ADDRESS`              | string                | `localhost:8888` | Listen address of the webhook.                                                                                                           |
| `METRICS_ADDRESS`              | string                | `:8080`          | Listen address of the prometheus metrics and health checks.                                                                              |

### Domain filters

Managed zones can be limited through domain filters. Limiting can be performed in either an inclusive or an exclusive
manner. Inclusive and exclusive filtering at the same time is supported. Filtering can be achieved either with a list
of [domain suffixes](#domain-suffixes), or via [regular expressions](#regular-expressions). Both methods cannot be used
at the same time. If both are set, domain suffixes are prioritized over regular expressions.

#### Domain suffixes

Domain suffixes are provided as a comma separated list. The two corresponding environment variables
`INCLUDE_DOMAIN_FILTER` and `EXCLUDE_DOMAIN_FILTER` can be set.

##### Example

```yaml
# values.yml
# [...]
provider:
  # [...]
  webhook:
    env:
      - name: INCLUDE_DOMAIN_FILTER
        value: >-
          example.com,
          example2.com,
          example3.com
      - name: HETZNER_TOKEN
        valueFrom:
          secretKeyRef:
            name: hetzner
            key: token
```

#### Regular expressions

Regular expressions are implemented with the [`regexp`](https://pkg.go.dev/regexp) package from the standard library.
The two corresponding environment variables `INCLUDE_REGEXP_DOMAIN_FILTER` and `EXCLUDE_REGEXP_DOMAIN_FILTER` can be
set.

##### Example

```yaml
# values.yml
# [...]
provider:
  # [...]
  webhook:
    env:
      - name: INCLUDE_REGEXP_DOMAIN_FILTER
        value: "example[0-9]*\\.com"
      - name: HETZNER_TOKEN
        valueFrom:
          secretKeyRef:
            name: hetzner
            key: token
```

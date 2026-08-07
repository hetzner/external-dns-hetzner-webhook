# Allow relative CNAME targets

This page describes how to use relative CNAME targets with the ExternalDNS Hetzner webhook.

## Default behaviour

ExternalDNS strips the trailing dot from CNAME targets
([external-dns#6145](https://github.com/kubernetes-sigs/external-dns/issues/6145)), which would make a target
pointing to another domain resolve within the zone of the record. The webhook therefore restores the trailing
dot and stores every CNAME target as an absolute name:

| Target from ExternalDNS | Stored value          |
| ----------------------- | --------------------- |
| `nginx.example.com`     | `nginx.example.com.`  |
| `nginx.absolute.com.`   | `nginx.absolute.com.` |
| `nginx.other.com`       | `nginx.other.com.`    |
| `nginx`                 | `nginx.`              |

This means relative CNAME targets cannot be used, a target such as `nginx` is stored as the absolute name
`nginx.` instead of being resolved within the zone of the record.

## Enable relative CNAME targets

If you rely on relative CNAME targets, set the
[`ALLOW_RELATIVE_CNAME_TARGETS`](../reference/configuration.md#supported-environment-variables) environment
variable to `true`:

```yaml
# values.yml
# [...]
provider:
  # [...]
  webhook:
    env:
      - name: ALLOW_RELATIVE_CNAME_TARGETS
        value: "true"
      - name: HETZNER_TOKEN
        valueFrom:
          secretKeyRef:
            name: hetzner
            key: token
```

Targets will then only made absolute if they end with a [public suffix](https://publicsuffix.org/), every other
targets are left untouched and stays relative to the zone of the record:

| Target from ExternalDNS | Stored value          |
| ----------------------- | --------------------- |
| `nginx.example.com`     | `nginx.example.com.`  |
| `nginx.absolute.com.`   | `nginx.absolute.com.` |
| `nginx.other.com`       | `nginx.other.com.`    |
| `nginx`                 | `nginx`               |
| `nginx.`                | `nginx`               |

## Known limitation

With `ALLOW_RELATIVE_CNAME_TARGETS` enabled, a relative target which ends with a public suffix cannot be expressed, because it is indistinguishable from an absolute target. You must use the absolute form instead. For the zone `example.com`, the relative target `embedded.other.de` has to be given as:

```yaml
external-dns.kubernetes.io/hostname: "www.example.com"
external-dns.kubernetes.io/target: "embedded.other.de.example.com."
```

> [!NOTE]
> We recommend always using an absolute target name (fully-qualified, ending with a `.`) to prevent confusion between relative and absolute targets, unless you know what you are doing.

# Changelog

## [v0.3.4](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.4)

[Compare to previous version](https://github.com/hetzner/external-dns-hetzner-webhook/compare/v0.3.3...v0.3.4)

### Trailing dots on CNAME targets are now restored

External DNS strips the trailing dot from CNAME targets ([kubernetes-sigs/external-dns#6145](https://github.com/kubernetes-sigs/external-dns/issues/6145)). The Hetzner DNS API treats a value without a trailing dot as relative to the zone, so a target such as `foo.other.com` in zone `example.com` was stored as a relative name and resolved to `foo.other.com.example.com`.

By default, the webhook now restores the trailing dot for any CNAME target.

If you rely on relative targets, you can set `ALLOW_RELATIVE_CNAME_TARGETS=t`. With this flag enabled, the webhook only restores the trailing dot when the CNAME target ends in a public suffix ([publicsuffix.org](https://publicsuffix.org/)). Otherwise, the target is stored relative to the zone and is unaffected. You can read more about this feature [here](docs/guides/allow-relative-cname-targets.md).

**Action required if you rely on a relative target.** With the default behavior (flag off), such a target is now treated as fully qualified. A target that previously resolved to `nginx.example.com` via the zone is now stored as `nginx.` and resolves to a different destination. Write the full name to keep the previous destination:

```diff
 metadata:
   annotations:
     external-dns.alpha.kubernetes.io/hostname: "www.example.com"
-    external-dns.alpha.kubernetes.io/target: "nginx"
+    external-dns.alpha.kubernetes.io/target: "nginx.example.com."
```

Only CNAME targets are adjusted, other record types keep their previous behavior.

### Bug Fixes

- CNAME resolution when External DNS strips trailing dots (#77) ([768d0e9](https://github.com/hetzner/external-dns-hetzner-webhook/commit/768d0e9cf8acb82ea14498b20adc906c8fd4d2f5))

## [v0.3.4-rc.0](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.4-rc.0)

[Compare to previous version](https://github.com/hetzner/external-dns-hetzner-webhook/compare/v0.3.3...v0.3.4-rc.0)

### CNAME targets are now stored as fully qualified names

External DNS strips the trailing dot from CNAME targets
([kubernetes-sigs/external-dns#6145](https://github.com/kubernetes-sigs/external-dns/issues/6145)).
The Hetzner DNS API treats a value without a trailing dot as relative to the zone, so a
target such as `foo.other.com` in zone `example.com` was stored as a relative name and
resolved to `foo.other.com.example.com`.

The webhook now restores the trailing dot for any CNAME target that ends in a public suffix
([publicsuffix.org](https://publicsuffix.org/)). Otherwise, they are stored relative to
the zone and are unaffected.

**Action required if you rely on a relative target that ends in a public suffix.** Such a
target is indistinguishable from a fully qualified name and is now treated as one. In zone
`example.com`, a target of `embedded.other.de` used to resolve to
`embedded.other.de.example.com` and now points to `embedded.other.de.` instead. Write the
full name to keep the previous destination:

```diff
 metadata:
   annotations:
     external-dns.alpha.kubernetes.io/hostname: "www.example.com"
-    external-dns.alpha.kubernetes.io/target: "embedded.other.de"
+    external-dns.alpha.kubernetes.io/target: "embedded.other.de.example.com"
```

Only CNAME targets are adjusted. MX and all other record types keep their previous
behavior.

### Bug Fixes

- CNAME resolution when External DNS strips trailing dots (#77) ([768d0e9](https://github.com/hetzner/external-dns-hetzner-webhook/commit/768d0e9cf8acb82ea14498b20adc906c8fd4d2f5))

## [v0.3.3](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.3)

### Bug Fixes

- ensure longest matching zone is selected (#112)

## [v0.3.2](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.2)

### Bug Fixes

- unset ttl when not configured (#69)

## [v0.3.1](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.1)

### Bug Fixes

- set API http client timeout to 15s (#62)

## [v0.3.0](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.3.0)

### Features

- increase fetch rrset page size (#42)

## [v0.2.0](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.2.0)

### New container image namespace

With this release, we moved our container image to [docker.io/hetzner/external-dns-hetzner-webhook](https://hub.docker.com/r/hetzner/external-dns-hetzner-webhook).

Users must update the External DNS Helm chart by updating the `provider.webhook.image.repository` value.

```diff
 # ...
 provider:
  webhook:
    image:
-     repository: docker.io/hetznercloud/external-dns-hetzner-webhook
+     repository: docker.io/hetzner/external-dns-hetzner-webhook
 # ...
```

Existing images in the old `hetznercloud` namespace will remain available, but new images will only be pushed to the new `hetzner` namespace.

### Features

- push image to hetzner docker namespace (#35)

## [v0.1.2](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.1.2)

### Bug Fixes

- wrong record name for apex domain (#27)

## [v0.1.1](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.1.1)

### Bug Fixes

- use nonroot user in container image (#16)

## [v0.1.0](https://github.com/hetzner/external-dns-hetzner-webhook/releases/tag/v0.1.0)

This release introduces the new [ExternalDNS](https://kubernetes-sigs.github.io/external-dns) webhook for Hetzner.

The webhook relies on the new [DNS API](https://docs.hetzner.cloud/reference/cloud#dns).

The DNS API is currently in **beta**, which will likely end on 10 November 2025. See the
[DNS Beta FAQ](https://docs.hetzner.com/networking/dns/faq/beta) for more details.

The webhook is currently experimental, breaking changes may occur within minor releases.

To get started, head to the [webhook documentation](https://github.com/hetzner/external-dns-hetzner-webhook/blob/main/docs).

### Features

- new external-dns webhook for Hetzner

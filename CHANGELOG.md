# Changelog

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

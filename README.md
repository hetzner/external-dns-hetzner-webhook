# external-dns-hetzner-webhook

![Maturity](https://img.shields.io/badge/maturity-experiment-orange) [![codecov](https://codecov.io/gh/hetzner/external-dns-hetzner-webhook/graph/badge.svg?token=XPN76GFHSE)](https://codecov.io/gh/hetzner/external-dns-hetzner-webhook)
![Tested K8s versions](https://img.shields.io/badge/dynamic/yaml?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhetzner%2Fexternal-dns-hetzner-webhook%2Fmain%2F.github%2Fworkflows%2Ftest.yml&query=%24.jobs.e2e.strategy.matrix.k8s&label=tested%20on%20k8s&color=326ce5&logo=kubernetes&logoColor=white)

Webhook which integrates ExternalDNS with the [Hetzner DNS API](https://docs.hetzner.cloud/reference/cloud#zones) via the [webhook provider API](https://kubernetes-sigs.github.io/external-dns/latest/docs/tutorials/webhook-provider/).

## Docs

- :rocket: See the [quick start guide](docs/guides/quickstart.md) to get you started.
- :book: See the [configuration reference](docs/reference/configuration.md) for the available configuration.

For more information, see the [documentation](docs/README.md).

# Quick start

This webhook is directly deployed via the [ExternalDNS Helm chart](https://kubernetes-sigs.github.io/external-dns/latest/charts/external-dns/) and runs as a sidecar. There is no separate chart for this webhook.

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/) as described in [this document](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/).

2. In the namespace of your ExternalDNS deployment create a secret containing your token:

```yaml
# secret.yml
apiVersion: v1
kind: Secret
metadata:
  name: hetzner
stringData:
  token: <HETZNER_TOKEN>
```

3. Configure the [ExternalDNS values](https://kubernetes-sigs.github.io/external-dns/latest/charts/external-dns/#values) to your needs and add the webhook provider:

```yaml
# values.yml
policy: sync
provider:
  name: webhook
  webhook:
    image:
      repository: docker.io/hetzner/external-dns-hetzner-webhook
      tag: v0.1.2 # x-releaser-pleaser-version
    env:
      - name: HETZNER_TOKEN
        valueFrom:
          secretKeyRef:
            name: hetzner
            key: token
```

4. Add the ExternalDNS Helm repository

```bash
helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
```

5. Install the Helm chart

```bash
helm upgrade --install external-dns external-dns/external-dns -f values.yml
```

> For more details about installing ExternalDNS, see the [ExternalDNS Helm chart documentation](https://kubernetes-sigs.github.io/external-dns/latest/charts/external-dns/).

module "infra" {
  source = "github.com/hetznercloud/kubernetes-dev-env//modules/infra?ref=v0.11.0"

  name         = "external-dns-hetzner-webhook-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  worker_count = 0

  hcloud_token = var.hetzner_token

  k3s_channel = var.k3s_channel

  # Share the generated files with the k8s state
  output_dir = abspath("${path.root}/../files")
}

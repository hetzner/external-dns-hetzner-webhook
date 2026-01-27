locals {
  kubeconfig_path    = abspath("${path.root}/files/kubeconfig.yaml")
  unboundconfig_path = abspath("${path.module}/unbound.conf")
}

data "local_sensitive_file" "kubeconfig" {
  depends_on = [module.dev]
  filename   = local.kubeconfig_path
}

provider "kubernetes" {
  config_path = data.local_sensitive_file.kubeconfig.filename
}

module "dev" {
  source = "github.com/hetznercloud/kubernetes-dev-env?ref=v0.10.0"

  name                 = "external-dns-hetzner-webhook-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token         = var.hetzner_token
  worker_count         = 0

  k3s_channel = var.k3s_channel
}

resource "kubernetes_secret_v1" "hetzner_token" {
  metadata {
    name      = "hetzner"
    namespace = "default"
  }

  data = {
    token = var.hetzner_token
  }
}

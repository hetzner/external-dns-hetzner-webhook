# The cluster itself is managed in the infra state, which must be applied first.
data "terraform_remote_state" "infra" {
  backend = "local"

  config = {
    path = abspath("${path.root}/../infra/terraform.tfstate")
  }
}

module "k8s" {
  source = "github.com/hetznercloud/kubernetes-dev-env//modules/k8s?ref=v0.11.0"

  hcloud_token = var.hetzner_token

  kubeconfig_path     = data.terraform_remote_state.infra.outputs.kubeconfig_filename
  hcloud_network_id   = data.terraform_remote_state.infra.outputs.network_id
  cluster_cidr        = data.terraform_remote_state.infra.outputs.cluster_cidr
  use_cloud_routes    = data.terraform_remote_state.infra.outputs.use_cloud_routes
  registry_service_ip = data.terraform_remote_state.infra.outputs.registry_service_ip

  # Share the generated files with the infra state
  output_dir = abspath("${path.root}/../files")
}

# The webhook reads the token from this Secret, see skaffold.yaml.
resource "kubernetes_secret_v1" "hetzner_token" {
  metadata {
    name      = "hetzner"
    namespace = "default"
  }

  data = {
    token = var.hetzner_token
  }
}

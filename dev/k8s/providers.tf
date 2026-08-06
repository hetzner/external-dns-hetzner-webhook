terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "< 4.0.0"
    }
  }
}

# Configured from the infra state, and never from an attribute of a resource
# managed in this state (see the k8s module).
provider "kubernetes" {
  config_path = data.terraform_remote_state.infra.outputs.kubeconfig_filename
}

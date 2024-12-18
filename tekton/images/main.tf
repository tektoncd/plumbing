terraform {
  required_providers {
    oci = {
        source = "chainguard-dev/oci" 
        version = "~> 0.0.10"
    }
    apko = {
        source = "chainguard-dev/apko"
        version = "~> 0.10.7"
    }
  }
}

provider "apko" {
  extra_repositories = ["https://packages.wolfi.dev/os"]
  extra_keyring      = ["https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"]
  extra_packages     = ["wolfi-baselayout"]
  default_archs      = ["x86_64", "aarch64"]
  default_annotations = {
    "org.opencontainers.image.authors" = "Tekton Authors <tekton-dev@googlegroups.com>"
  }
}

variable "target_repository" {
  description = "The docker repo into which the image and attestations should be published."
}

module "ko" {
    source = "./ko"
    target_repository = "${var.target_repository}/ko"
}

module "ko-gcloud" {
    source = "./ko-gcloud"
    target_repository = "${var.target_repository}/ko-gcloud"
}

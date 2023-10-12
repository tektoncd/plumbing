terraform {
  required_providers {
    apko = {
      source = "chainguard-dev/apko"
    }
    oci = {
      source = "chainguard-dev/oci"
    }
  }
}

variable "target_repository" {
  description = "The docker repo into which the image and attestations should be published."
}

module "image" {
  source  = "chainguard-dev/apko/publisher"
  version = "0.0.9"

  target_repository = var.target_repository
  config = file("${path.module}/apko.yaml")
  default_annotations = {
    "org.opencontainers.image.url": "https://github.com/tektoncd/plumbing/tree/main/tekton/images/ko"
  }
}

resource "oci_tag" "latest" {
  digest_ref = module.image.image_ref
  tag        = "latest-wolfi"
}

output "image_ref" {
  value = oci_tag.latest.tagged_ref
}
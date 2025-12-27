# Registry Module - OCI Container Registry
# Free Tier: 500MB total storage

variable "compartment_ocid" {
  type = string
}

variable "repository_name" {
  type    = string
  default = "playground"
}

variable "is_public" {
  type    = bool
  default = false
}

# Container Repository
resource "oci_artifacts_container_repository" "main" {
  compartment_id = var.compartment_ocid
  display_name   = var.repository_name
  is_public      = var.is_public
  is_immutable   = false

  readme {
    content = <<-EOT
      # Container Registry for ${var.repository_name}

      ## ⚠️ Free Tier Limit: 500MB Total

      ### Best Practices
      - Use semantic versioning (v1.0.0), not "latest"
      - Use multi-stage Docker builds
      - Keep only last 5 images per tag
      - Delete debug/test images
    EOT
    format  = "TEXT_MARKDOWN"
  }
}

# Outputs
output "repository_id" {
  value = oci_artifacts_container_repository.main.id
}

output "repository_name" {
  value = oci_artifacts_container_repository.main.display_name
}

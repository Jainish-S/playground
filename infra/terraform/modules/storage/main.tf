# Storage Module - OCI Object Storage
# Free Tier: 20GB (10GB Standard + 10GB Archive)

variable "compartment_ocid" {
  type = string
}

variable "bucket_name" {
  type    = string
  default = "playground-data"
}

# Get namespace
data "oci_objectstorage_namespace" "main" {
  compartment_id = var.compartment_ocid
}

# Object Storage Bucket
resource "oci_objectstorage_bucket" "main" {
  compartment_id = var.compartment_ocid
  name           = var.bucket_name
  namespace      = data.oci_objectstorage_namespace.main.namespace
  access_type    = "NoPublicAccess"
  storage_tier   = "Standard"
  versioning     = "Disabled"
}

# Outputs
output "bucket_name" {
  value = oci_objectstorage_bucket.main.name
}

output "namespace" {
  value = data.oci_objectstorage_namespace.main.namespace
}

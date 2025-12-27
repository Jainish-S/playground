# Backend Configuration
# Currently using local state. 
# TODO: Migrate to OCI Object Storage for remote state.

terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}

# =============================================================================
# FUTURE: OCI Object Storage Backend
# =============================================================================
# Uncomment below and comment out local backend when ready:
#
# terraform {
#   backend "http" {
#     address        = "https://objectstorage.<region>.oraclecloud.com/p/<pre-auth-token>/n/<namespace>/b/<bucket>/o/terraform.tfstate"
#     update_method  = "PUT"
#   }
# }
#
# Alternative: Use S3-compatible backend with OCI
# terraform {
#   backend "s3" {
#     bucket                      = "terraform-state"
#     key                         = "playground/dev/terraform.tfstate"
#     region                      = "ap-mumbai-1"
#     endpoint                    = "https://<namespace>.compat.objectstorage.<region>.oraclecloud.com"
#     skip_region_validation      = true
#     skip_credentials_validation = true
#     skip_metadata_api_check     = true
#     force_path_style            = true
#   }
# }

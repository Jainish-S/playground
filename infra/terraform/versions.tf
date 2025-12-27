terraform {
  required_version = ">= 1.0.0"

  required_providers {
    oci = {
      source  = "oracle/oci"
      version = ">= 5.0.0"
    }
    twingate = {
      source  = "Twingate/twingate"
      version = "~> 3.6"
    }
  }
}

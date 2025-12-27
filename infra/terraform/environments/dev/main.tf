# Dev Environment - Module Composition
# Composes all infrastructure modules for development

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

# OCI Provider
provider "oci" {
  tenancy_ocid     = var.tenancy_ocid
  user_ocid        = var.user_ocid
  fingerprint      = var.fingerprint
  private_key_path = var.private_key_path
  region           = var.region
}

# Twingate Provider (conditional)
provider "twingate" {
  api_token = var.twingate_api_token
  network   = var.twingate_network
}

# Network Module
module "network" {
  source = "../../modules/network"

  compartment_ocid = var.compartment_ocid
  ssh_allowed_cidr = var.ssh_allowed_cidr
}

# OKE Cluster Module
module "oke" {
  source = "../../modules/oke"

  compartment_ocid = var.compartment_ocid
  vcn_id           = module.network.vcn_id
  subnet_id        = module.network.subnet_id
  ssh_public_key   = var.ssh_public_key
}

# Container Registry Module
module "registry" {
  source = "../../modules/registry"

  compartment_ocid = var.compartment_ocid
  repository_name  = "playground"
}

# Object Storage Module
module "storage" {
  source = "../../modules/storage"

  compartment_ocid = var.compartment_ocid
  bucket_name      = "playground-data"
}

# Twingate Module (conditional)
module "twingate" {
  source = "../../modules/twingate"
  count  = var.enable_twingate ? 1 : 0

  twingate_api_token = var.twingate_api_token
  twingate_network   = var.twingate_network
}

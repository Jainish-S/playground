# OKE Module - Kubernetes Cluster and Node Pool
# Optimized for OCI Free Tier (4 OCPUs, 24GB RAM)

variable "compartment_ocid" {
  type = string
}

variable "vcn_id" {
  type = string
}

variable "subnet_id" {
  type = string
}

variable "ssh_public_key" {
  type = string
}

variable "kubernetes_version" {
  type    = string
  default = "v1.34.1"
}

variable "node_pool_size" {
  type    = number
  default = 2
}

variable "node_ocpus" {
  type    = number
  default = 2
}

variable "node_memory_gb" {
  type    = number
  default = 12
}

# Data: Availability Domains
data "oci_identity_availability_domains" "ads" {
  compartment_id = var.compartment_ocid
}

# OKE Cluster
resource "oci_containerengine_cluster" "main" {
  compartment_id     = var.compartment_ocid
  kubernetes_version = var.kubernetes_version
  name               = "k8s-cluster"
  vcn_id             = var.vcn_id

  lifecycle {
    create_before_destroy = false
  }

  endpoint_config {
    is_public_ip_enabled = true
    subnet_id            = var.subnet_id
  }

  options {
    add_ons {
      is_kubernetes_dashboard_enabled = false
      is_tiller_enabled               = false
    }
    service_lb_config {}
  }
}

# Data: Node Pool Options (for image ID)
data "oci_containerengine_node_pool_option" "main" {
  node_pool_option_id = oci_containerengine_cluster.main.id
  compartment_id      = var.compartment_ocid
}

# Node Pool - ARM64 (Ampere A1)
resource "oci_containerengine_node_pool" "main" {
  cluster_id         = oci_containerengine_cluster.main.id
  compartment_id     = var.compartment_ocid
  kubernetes_version = oci_containerengine_cluster.main.kubernetes_version
  name               = "k8s-node-pool"
  node_shape         = "VM.Standard.A1.Flex"

  node_shape_config {
    memory_in_gbs = var.node_memory_gb
    ocpus         = var.node_ocpus
  }

  node_source_details {
    image_id    = lookup(data.oci_containerengine_node_pool_option.main.sources[0], "image_id")
    source_type = "IMAGE"
  }

  node_config_details {
    placement_configs {
      availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
      subnet_id           = var.subnet_id
    }
    size = var.node_pool_size
  }

  ssh_public_key = var.ssh_public_key

  initial_node_labels {
    key   = "role"
    value = "worker"
  }
}

# Outputs
output "cluster_id" {
  value = oci_containerengine_cluster.main.id
}

output "node_pool_id" {
  value = oci_containerengine_node_pool.main.id
}

output "kubeconfig_command" {
  value = "oci ce cluster create-kubeconfig --cluster-id ${oci_containerengine_cluster.main.id} --file $HOME/.kube/config --region ${split(".", oci_containerengine_cluster.main.id)[3]} --token-version 2.0.0"
}

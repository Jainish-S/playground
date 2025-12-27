# Twingate Module - Zero Trust Network Access
# Manages remote network, connectors, and resources
# Connectors deployed via K8s route traffic to internal services

terraform {
  required_providers {
    twingate = {
      source  = "Twingate/twingate"
      version = "~> 3.6"
    }
  }
}

variable "twingate_api_token" {
  type      = string
  sensitive = true
}

variable "twingate_network" {
  type = string
}

variable "remote_network_name" {
  type    = string
  default = "OCI-Kubernetes"
}

variable "k8s_api_private_ip" {
  description = "Private IP of K8s API server (from kubectl cluster-info)"
  type        = string
  default     = "10.0.1.104"
}

# Remote Network
resource "twingate_remote_network" "main" {
  name     = var.remote_network_name
  location = "OTHER"
}

# Connectors (tokens used to deploy pods in K8s)
resource "twingate_connector" "primary" {
  remote_network_id = twingate_remote_network.main.id
  name              = "OKE-Primary-Connector"
}

resource "twingate_connector" "secondary" {
  remote_network_id = twingate_remote_network.main.id
  name              = "OKE-Secondary-Connector"
}

resource "twingate_connector_tokens" "primary" {
  connector_id = twingate_connector.primary.id
}

resource "twingate_connector_tokens" "secondary" {
  connector_id = twingate_connector.secondary.id
}

# Access Groups
resource "twingate_group" "admins" {
  name = "Kubernetes Administrators"
}

resource "twingate_group" "developers" {
  name = "Developers"
}

# =============================================================================
# Resources - Internal K8s Services (accessed via connector)
# =============================================================================

# Kubernetes API (private IP - for kubectl access via Twingate)
resource "twingate_resource" "k8s_api" {
  name              = "Kubernetes API Server"
  address           = var.k8s_api_private_ip
  remote_network_id = twingate_remote_network.main.id

  protocols = {
    allow_icmp = false
    tcp = {
      policy = "RESTRICTED"
      ports  = ["6443"]
    }
    udp = {
      policy = "DENY_ALL"
    }
  }

  access_group {
    group_id = twingate_group.admins.id
  }
}

# Grafana Dashboard
resource "twingate_resource" "grafana" {
  name              = "Grafana Dashboard"
  address           = "grafana.observability.svc.cluster.local"
  remote_network_id = twingate_remote_network.main.id

  access_group {
    group_id = twingate_group.admins.id
  }
  access_group {
    group_id = twingate_group.developers.id
  }
}

# Prometheus
resource "twingate_resource" "prometheus" {
  name              = "Prometheus"
  address           = "prometheus.observability.svc.cluster.local"
  remote_network_id = twingate_remote_network.main.id

  access_group {
    group_id = twingate_group.admins.id
  }
  access_group {
    group_id = twingate_group.developers.id
  }
}

# Nginx Ingress
resource "twingate_resource" "ingress" {
  name              = "Nginx Ingress"
  address           = "ingress-nginx-controller.ingress-nginx.svc.cluster.local"
  remote_network_id = twingate_remote_network.main.id

  access_group {
    group_id = twingate_group.admins.id
  }
  access_group {
    group_id = twingate_group.developers.id
  }
}

# Outputs
output "remote_network_id" {
  value = twingate_remote_network.main.id
}

output "primary_connector_tokens" {
  value     = twingate_connector_tokens.primary
  sensitive = true
}

output "secondary_connector_tokens" {
  value     = twingate_connector_tokens.secondary
  sensitive = true
}

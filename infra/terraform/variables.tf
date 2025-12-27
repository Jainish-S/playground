# =============================================================================
# OCI Authentication
# =============================================================================

variable "tenancy_ocid" {
  description = "OCI Tenancy OCID"
  type        = string
}

variable "user_ocid" {
  description = "OCI User OCID"
  type        = string
}

variable "fingerprint" {
  description = "OCI API key fingerprint"
  type        = string
}

variable "private_key_path" {
  description = "Path to OCI API private key file"
  type        = string
}

variable "region" {
  description = "OCI region (e.g., ap-mumbai-1)"
  type        = string
}

variable "compartment_ocid" {
  description = "OCI Compartment OCID for resource creation"
  type        = string
}

# =============================================================================
# SSH Access
# =============================================================================

variable "ssh_public_key" {
  description = "SSH public key for node access"
  type        = string
}

variable "ssh_allowed_cidr" {
  description = "CIDR block allowed for SSH. Use YOUR IP (x.x.x.x/32) for security."
  type        = string
  default     = "0.0.0.0/0"

  validation {
    condition     = can(cidrhost(var.ssh_allowed_cidr, 0))
    error_message = "Must be a valid CIDR block (e.g., 1.2.3.4/32)."
  }
}

# =============================================================================
# Twingate
# =============================================================================

variable "twingate_api_token" {
  description = "Twingate API token (from Settings → API)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "twingate_network" {
  description = "Twingate network name (your-org.twingate.com → 'your-org')"
  type        = string
  default     = ""
}

variable "enable_twingate" {
  description = "Enable Twingate resources creation"
  type        = bool
  default     = false
}

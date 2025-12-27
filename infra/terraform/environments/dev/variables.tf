# Variables for Dev Environment
# Inherits from root variables.tf

variable "tenancy_ocid" {
  type = string
}

variable "user_ocid" {
  type = string
}

variable "fingerprint" {
  type = string
}

variable "private_key_path" {
  type = string
}

variable "region" {
  type = string
}

variable "compartment_ocid" {
  type = string
}

variable "ssh_public_key" {
  type = string
}

variable "ssh_allowed_cidr" {
  type    = string
  default = "0.0.0.0/0"
}

variable "twingate_api_token" {
  type      = string
  sensitive = true
  default   = ""
}

variable "twingate_network" {
  type    = string
  default = ""
}

variable "enable_twingate" {
  type    = bool
  default = false
}

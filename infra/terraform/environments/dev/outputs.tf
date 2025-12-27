# Outputs for Dev Environment

output "cluster_id" {
  value = module.oke.cluster_id
}

output "kubeconfig_command" {
  value = module.oke.kubeconfig_command
}

output "registry_name" {
  value = module.registry.repository_name
}

output "storage_namespace" {
  value = module.storage.namespace
}

output "twingate_primary_tokens" {
  value     = var.enable_twingate ? module.twingate[0].primary_connector_tokens : null
  sensitive = true
}

output "twingate_secondary_tokens" {
  value     = var.enable_twingate ? module.twingate[0].secondary_connector_tokens : null
  sensitive = true
}

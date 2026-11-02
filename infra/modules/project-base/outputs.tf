output "workload_identity_provider" {
  value = module.github_wif.workload_identity_provider
}

output "ci_service_account_email" {
  value = module.github_wif.ci_service_account_email
}

output "image_prefix" {
  value = module.artifact_registry.image_prefix
}

output "project_number" {
  value = data.google_project.this.number
}

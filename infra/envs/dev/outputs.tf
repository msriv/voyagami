output "workload_identity_provider" {
  value = module.github_wif.workload_identity_provider
}

output "ci_service_account_email" {
  value = module.github_wif.ci_service_account_email
}

output "api_url" {
  value = module.api.url
}

output "image_prefix" {
  value = module.artifact_registry.image_prefix
}

output "api_runtime_service_account" {
  value = module.api_runtime.email
}

output "media_bucket" {
  value = module.media_bucket.name
}

output "pubsub_topics" {
  value = module.topics.topic_ids
}

output "workload_identity_provider" {
  value = module.base.workload_identity_provider
}

output "ci_service_account_email" {
  value = module.base.ci_service_account_email
}

output "image_prefix" {
  value = module.base.image_prefix
}

output "api_url" {
  value = module.stack.api_url
}

output "api_runtime_service_account" {
  value = module.stack.api_runtime_service_account
}

output "media_bucket" {
  value = module.stack.media_bucket
}

output "pubsub_topics" {
  value = module.stack.pubsub_topics
}

output "secret_ids" {
  value = module.stack.secret_ids
}

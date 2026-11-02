output "api_url" {
  value = module.api.url
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

output "secret_ids" {
  description = "Every secret this stack reads, for apply.sh."
  value       = values(local.secret_env)
}

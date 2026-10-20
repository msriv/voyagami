output "topic_ids" {
  description = "Topic short name to full topic ID."
  value       = { for k, t in google_pubsub_topic.this : k => t.id }
}

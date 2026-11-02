output "topic_name" {
  value = google_pubsub_topic.events.name
}

output "dataset_id" {
  value = google_bigquery_dataset.analytics.dataset_id
}

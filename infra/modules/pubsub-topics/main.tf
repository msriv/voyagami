resource "google_pubsub_topic" "this" {
  for_each = toset(var.topics)

  project = var.project_id
  name    = "${var.name_prefix}${each.value}"

  message_retention_duration = var.message_retention
}

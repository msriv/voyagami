resource "google_firestore_database" "this" {
  project     = var.project_id
  name        = var.name
  location_id = var.location_id
  type        = "FIRESTORE_NATIVE"

  # Point-in-time recovery keeps 7 days of versions (prod only; it costs storage).
  point_in_time_recovery_enablement = var.backups ? "POINT_IN_TIME_RECOVERY_ENABLED" : "POINT_IN_TIME_RECOVERY_DISABLED"

  # A dropped database loses all data. Terraform must not do it silently.
  deletion_policy = "ABANDON"
}

# Daily backup kept 14 days: recovery point at most 24 h old (VOY-74 RPO).
resource "google_firestore_backup_schedule" "daily" {
  count = var.backups ? 1 : 0

  project   = var.project_id
  database  = google_firestore_database.this.name
  retention = "1209600s"

  daily_recurrence {}
}

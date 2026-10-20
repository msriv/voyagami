resource "google_firestore_database" "this" {
  project     = var.project_id
  name        = var.name
  location_id = var.location_id
  type        = "FIRESTORE_NATIVE"

  # A dropped database loses all data. Terraform must not do it silently.
  deletion_policy = "ABANDON"
}

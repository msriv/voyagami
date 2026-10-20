# Creates empty secrets only. Values are added by hand or by a later ticket,
# never in Terraform, so they stay out of state and the repo.
resource "google_secret_manager_secret" "this" {
  for_each = toset(var.secret_ids)

  project   = var.project_id
  secret_id = each.value

  replication {
    auto {}
  }
}

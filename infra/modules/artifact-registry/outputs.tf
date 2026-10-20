output "repository_id" {
  value = google_artifact_registry_repository.this.repository_id
}

output "image_prefix" {
  description = "Prefix for image names, for example <prefix>/api:tag."
  value       = "${var.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.this.repository_id}"
}

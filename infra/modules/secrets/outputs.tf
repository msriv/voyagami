output "secret_ids" {
  description = "Secret short name to resource ID."
  value       = { for k, s in google_secret_manager_secret.this : k => s.id }
}

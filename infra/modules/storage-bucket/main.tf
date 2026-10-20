# Private bucket. Clients upload and download only through signed URLs.
resource "google_storage_bucket" "this" {
  project  = var.project_id
  name     = var.name
  location = var.location

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = var.force_destroy

  cors {
    origin          = var.cors_origins
    method          = ["GET", "HEAD", "PUT"]
    response_header = ["Content-Type", "Content-MD5", "x-goog-meta-*"]
    max_age_seconds = 3600
  }

  lifecycle_rule {
    condition {
      age = 1
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }
}

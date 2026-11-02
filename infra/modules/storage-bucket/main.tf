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

  versioning {
    enabled = var.versioning
  }

  # With versioning on, replaced or deleted files are kept 30 days, then removed.
  dynamic "lifecycle_rule" {
    for_each = var.versioning ? [1] : []
    content {
      condition {
        days_since_noncurrent_time = 30
      }
      action {
        type = "Delete"
      }
    }
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

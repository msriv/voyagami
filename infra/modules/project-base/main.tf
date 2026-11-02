# Project-wide pieces: APIs, CI identity, image registry and the budget alert.
# One per GCP project. Environments that share a project (staging inside
# voyagami-dev) use only app-stack.

module "apis" {
  source     = "../project-apis"
  project_id = var.project_id

  apis = [
    "aiplatform.googleapis.com",
    "artifactregistry.googleapis.com",
    "billingbudgets.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "cloudscheduler.googleapis.com",
    "fcm.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "identitytoolkit.googleapis.com",
    "logging.googleapis.com",
    "maps-android-backend.googleapis.com",
    "maps-ios-backend.googleapis.com",
    "monitoring.googleapis.com",
    "places.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com",
  ]
}

module "github_wif" {
  source            = "../github-wif"
  project_id        = var.project_id
  github_repository = var.github_repository
  state_bucket      = var.state_bucket
  ci_roles          = var.ci_roles

  depends_on = [module.apis]
}

module "artifact_registry" {
  source        = "../artifact-registry"
  project_id    = var.project_id
  location      = var.region
  repository_id = "voyagami"
  description   = "Voyagami container images"

  depends_on = [module.apis]
}

# What CI needs to push an image.
resource "google_artifact_registry_repository_iam_member" "ci_push" {
  project    = var.project_id
  location   = var.region
  repository = module.artifact_registry.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${module.github_wif.ci_service_account_email}"
}

data "google_project" "this" {
  project_id = var.project_id
}

# Alerts only, never a spend cap. Emails go to the billing account's admins.
# The amount is in the billing account's own currency (no currency code here).
resource "google_billing_budget" "monthly" {
  count = var.budget_amount > 0 ? 1 : 0

  billing_account = var.billing_account
  display_name    = "${var.project_id} monthly"

  budget_filter {
    projects = ["projects/${data.google_project.this.number}"]
  }

  amount {
    specified_amount {
      units = tostring(var.budget_amount)
    }
  }

  dynamic "threshold_rules" {
    for_each = [0.5, 0.8, 1.0]
    content {
      threshold_percent = threshold_rules.value
    }
  }

  depends_on = [module.apis]
}

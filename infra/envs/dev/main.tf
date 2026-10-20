module "apis" {
  source     = "../../modules/project-apis"
  project_id = var.project_id

  apis = [
    "aiplatform.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "fcm.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "identitytoolkit.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "pubsub.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com",
  ]
}

module "github_wif" {
  source            = "../../modules/github-wif"
  project_id        = var.project_id
  github_repository = var.github_repository
  state_bucket      = var.state_bucket

  # Deploy roles arrive with the Cloud Run, Firestore and Pub/Sub tickets
  # (VOY-89, VOY-90). Start small; widen only when a ticket needs it.
  ci_roles = [
    "roles/viewer",
  ]

  depends_on = [module.apis]
}

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
    "roles/run.developer",
  ]

  depends_on = [module.apis]
}

module "artifact_registry" {
  source        = "../../modules/artifact-registry"
  project_id    = var.project_id
  location      = var.region
  repository_id = "voyagami"
  description   = "Voyagami container images (dev)"

  depends_on = [module.apis]
}

# Runtime identity of the API. Starts with logging only; VOY-90 adds Firestore,
# Pub/Sub and Secret Manager access as those resources appear.
module "api_runtime" {
  source       = "../../modules/service-account"
  project_id   = var.project_id
  account_id   = "api-runtime"
  display_name = "API runtime (Cloud Run)"

  roles = [
    "roles/logging.logWriter",
  ]

  depends_on = [module.apis]
}

module "api" {
  source                = "../../modules/cloud-run"
  project_id            = var.project_id
  region                = var.region
  name                  = "voyagami-api"
  service_account_email = module.api_runtime.email
  min_instances         = 0
  allow_unauthenticated = true

  depends_on = [module.apis]
}

# What CI needs to push an image and deploy the service.
resource "google_artifact_registry_repository_iam_member" "ci_push" {
  project    = var.project_id
  location   = var.region
  repository = module.artifact_registry.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${module.github_wif.ci_service_account_email}"
}

resource "google_service_account_iam_member" "ci_act_as_runtime" {
  service_account_id = module.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${module.github_wif.ci_service_account_email}"
}

module "firestore" {
  source      = "../../modules/firestore"
  project_id  = var.project_id
  location_id = var.region

  depends_on = [module.apis]
}

module "topics" {
  source     = "../../modules/pubsub-topics"
  project_id = var.project_id
  topics     = ["notifications", "recap", "place-ingest"]

  depends_on = [module.apis]
}

module "media_bucket" {
  source     = "../../modules/storage-bucket"
  project_id = var.project_id
  name       = "${var.project_id}-media"
  location   = var.region

  # Dev only: lets a throwaway dev project be torn down cleanly.
  force_destroy = true

  depends_on = [module.apis]
}

module "secrets" {
  source     = "../../modules/secrets"
  project_id = var.project_id

  # Empty containers. Add the value with:
  #   printf '%s' "$KEY" | gcloud secrets versions add google-maps-server-key --data-file=-
  secret_ids = ["google-maps-server-key"]

  depends_on = [module.apis]
}

# What the API runtime account may touch. Least privilege, resource by resource.
resource "google_project_iam_member" "api_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${module.api_runtime.email}"
}

resource "google_pubsub_topic_iam_member" "api_publish" {
  for_each = module.topics.topic_ids

  project = var.project_id
  topic   = each.value
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${module.api_runtime.email}"
}

resource "google_storage_bucket_iam_member" "api_media" {
  bucket = module.media_bucket.name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${module.api_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "api_secrets" {
  for_each = module.secrets.secret_ids

  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${module.api_runtime.email}"
}

# Signed upload and download URLs are signed with the runtime account's own key.
resource "google_service_account_iam_member" "api_sign_urls" {
  service_account_id = module.api_runtime.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${module.api_runtime.email}"
}

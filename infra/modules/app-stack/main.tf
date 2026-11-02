# One running copy of Voyagami: API service, its identity, data stores, secrets
# and scheduled jobs. Every name takes var.prefix, so staging can live in the dev
# project next to dev ("" for dev and prod, "staging-" for staging).

locals {
  # Secrets that apply.sh fills with a random value when they are empty.
  generated_secrets = ["auth-secret", "jobs-secret", "revenuecat-webhook-secret"]
  # Secrets a person must fill (apply.sh stops and says how).
  manual_secrets = ["google-maps-server-key"]

  secret_env = {
    AUTH_SECRET               = "${var.prefix}auth-secret"
    JOBS_SECRET               = "${var.prefix}jobs-secret"
    REVENUECAT_WEBHOOK_SECRET = "${var.prefix}revenuecat-webhook-secret"
    PLACES_API_KEY            = "${var.prefix}google-maps-server-key"
  }
}

# Runtime identity of the API. Least privilege, resource by resource below.
module "api_runtime" {
  source       = "../service-account"
  project_id   = var.project_id
  account_id   = "${var.prefix}api-runtime"
  display_name = "API runtime (Cloud Run)"
  roles        = ["roles/logging.logWriter"]
}

module "secrets" {
  source     = "../secrets"
  project_id = var.project_id

  # Empty containers. Values never go through Terraform: infra/scripts/apply.sh
  # adds random ones, and the Maps key is added by hand (see infra/README.md).
  secret_ids = [for s in concat(local.manual_secrets, local.generated_secrets) : "${var.prefix}${s}"]
}

module "api" {
  source                = "../cloud-run"
  project_id            = var.project_id
  region                = var.region
  name                  = "${var.prefix}voyagami-api"
  service_account_email = module.api_runtime.email
  min_instances         = var.min_instances
  allow_unauthenticated = true

  env = {
    APP_ENV         = var.env
    GCP_PROJECT     = var.project_id
    VERTEX_LOCATION = "us-central1" # booking import calls Gemini here; documents leave the home region
  }

  # Cloud Run cannot start a revision whose secret has no value, so apply.sh
  # fills every secret before this is applied.
  secret_env = local.secret_env

  depends_on = [google_secret_manager_secret_iam_member.api_secrets]
}

# Lets CI deploy new revisions that run as the runtime account.
resource "google_service_account_iam_member" "ci_act_as_runtime" {
  service_account_id = module.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.ci_service_account_email}"
}

module "firestore" {
  source      = "../firestore"
  project_id  = var.project_id
  name        = var.firestore_database
  location_id = var.region
}

module "topics" {
  source      = "../pubsub-topics"
  project_id  = var.project_id
  name_prefix = var.prefix
  topics      = ["notifications", "recap", "place-ingest"]
}

module "media_bucket" {
  source        = "../storage-bucket"
  project_id    = var.project_id
  name          = "${var.project_id}-${var.prefix}media"
  location      = var.region
  force_destroy = var.env == "dev" # lets a throwaway dev project be torn down cleanly
}

resource "google_project_iam_member" "api_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${module.api_runtime.email}"
}

# Lets the API call Gemini on Vertex (booking import) with its own identity.
resource "google_project_iam_member" "api_vertex" {
  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${module.api_runtime.email}"
}

# Lets the API send push messages through Firebase Cloud Messaging.
resource "google_project_iam_member" "api_fcm" {
  project = var.project_id
  role    = "roles/firebasecloudmessaging.admin"
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

# The scheduler sends the jobs secret as a bearer token. Reading it here puts the
# value in Terraform state (private, versioned bucket); the API checks it.
data "google_secret_manager_secret_version" "jobs" {
  project = var.project_id
  secret  = "${var.prefix}jobs-secret"

  depends_on = [module.secrets]
}

locals {
  jobs = {
    "trial-reminders"   = "0 */3 * * *" # "your trial ends in 2 days" push; safe to repeat
    "account-deletions" = "0 3 * * *"   # deletes accounts whose 30 days are over
  }
}

resource "google_cloud_scheduler_job" "jobs" {
  for_each = local.jobs

  project   = var.project_id
  region    = var.region
  name      = "${var.prefix}${each.key}"
  schedule  = each.value
  time_zone = "Etc/UTC"

  http_target {
    http_method = "POST"
    uri         = "${module.api.url}/v1/jobs/${each.key}"
    headers = {
      Authorization = "Bearer ${data.google_secret_manager_secret_version.jobs.secret_data}"
    }
  }
}

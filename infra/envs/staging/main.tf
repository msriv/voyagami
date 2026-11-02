# Staging lives inside the dev project (decided 2026-10-20): its own API service,
# Firestore database, topics, bucket, secrets and runtime account, all prefixed
# "staging-". APIs, the image registry and the CI identity come from envs/dev,
# so apply dev first.
module "stack" {
  source                   = "../../modules/app-stack"
  project_id               = var.project_id
  region                   = var.region
  env                      = "staging"
  prefix                   = "staging-"
  firestore_database       = "staging"
  ci_service_account_email = var.ci_service_account_email
}

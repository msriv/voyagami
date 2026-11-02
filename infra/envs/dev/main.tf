module "base" {
  source            = "../../modules/project-base"
  project_id        = var.project_id
  region            = var.region
  github_repository = var.github_repository
  state_bucket      = var.state_bucket
  billing_account   = var.billing_account
  budget_amount     = var.budget_amount
}

module "stack" {
  source                   = "../../modules/app-stack"
  project_id               = var.project_id
  region                   = var.region
  env                      = "dev"
  alert_email              = var.alert_email
  ci_service_account_email = module.base.ci_service_account_email

  depends_on = [module.base]
}

# Dev was one flat root until VOY-225. These keep the live resources in place.
moved {
  from = module.apis
  to   = module.base.module.apis
}
moved {
  from = module.github_wif
  to   = module.base.module.github_wif
}
moved {
  from = module.artifact_registry
  to   = module.base.module.artifact_registry
}
moved {
  from = google_artifact_registry_repository_iam_member.ci_push
  to   = module.base.google_artifact_registry_repository_iam_member.ci_push
}
moved {
  from = module.api_runtime
  to   = module.stack.module.api_runtime
}
moved {
  from = module.api
  to   = module.stack.module.api
}
moved {
  from = google_service_account_iam_member.ci_act_as_runtime
  to   = module.stack.google_service_account_iam_member.ci_act_as_runtime
}
moved {
  from = module.firestore
  to   = module.stack.module.firestore
}
moved {
  from = module.topics
  to   = module.stack.module.topics
}
moved {
  from = module.media_bucket
  to   = module.stack.module.media_bucket
}
moved {
  from = module.secrets
  to   = module.stack.module.secrets
}
moved {
  from = google_project_iam_member.api_firestore
  to   = module.stack.google_project_iam_member.api_firestore
}
moved {
  from = google_project_iam_member.api_vertex
  to   = module.stack.google_project_iam_member.api_vertex
}
moved {
  from = google_project_iam_member.api_fcm
  to   = module.stack.google_project_iam_member.api_fcm
}
moved {
  from = google_pubsub_topic_iam_member.api_publish
  to   = module.stack.google_pubsub_topic_iam_member.api_publish
}
moved {
  from = google_storage_bucket_iam_member.api_media
  to   = module.stack.google_storage_bucket_iam_member.api_media
}
moved {
  from = google_secret_manager_secret_iam_member.api_secrets
  to   = module.stack.google_secret_manager_secret_iam_member.api_secrets
}
moved {
  from = google_service_account_iam_member.api_sign_urls
  to   = module.stack.google_service_account_iam_member.api_sign_urls
}

# Production: its own project, so real users, push, IAM, quotas and budget stay
# apart from dev. The project itself is created by hand once (see infra/README.md).
module "base" {
  source            = "../../modules/project-base"
  project_id        = var.project_id
  region            = var.region
  github_repository = var.github_repository
  state_bucket      = "${var.project_id}-tfstate"
  billing_account   = var.billing_account
  budget_amount     = var.budget_amount
}

module "stack" {
  source                   = "../../modules/app-stack"
  project_id               = var.project_id
  region                   = var.region
  env                      = "prod"
  min_instances            = 1 # no cold starts for real users
  backups                  = true
  alert_email              = var.alert_email
  ci_service_account_email = module.base.ci_service_account_email

  depends_on = [module.base]
}

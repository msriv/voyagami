variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "env" {
  description = "dev, staging or prod. Sets APP_ENV on the API."
  type        = string
}

variable "prefix" {
  description = "Prefix for every name. \"\" for dev and prod, \"staging-\" for staging."
  type        = string
  default     = ""
}

variable "firestore_database" {
  description = "Firestore database ID. \"(default)\" except for staging."
  type        = string
  default     = "(default)"
}

variable "min_instances" {
  description = "Cloud Run minimum instances. 0 scales to zero."
  type        = number
  default     = 0
}

variable "ci_service_account_email" {
  description = "CI account that deploys the API (from project-base)."
  type        = string
}

variable "alert_email" {
  description = "Where alerts go. apply.sh passes the signed-in gcloud account. Empty means no emails."
  type        = string
  default     = ""
}

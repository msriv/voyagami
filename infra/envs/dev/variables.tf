variable "project_id" {
  description = "GCP project ID for dev."
  type        = string
  default     = "voyagami-dev-512208"
}

variable "region" {
  type    = string
  default = "asia-south1"
}

variable "github_repository" {
  type    = string
  default = "msriv/voyagami"
}

variable "state_bucket" {
  type    = string
  default = "voyagami-dev-512208-tfstate"
}

variable "billing_account" {
  description = "Billing account ID. apply.sh passes it; empty means no budget."
  type        = string
  default     = ""
}

variable "budget_amount" {
  description = "Monthly budget alert, in the billing account's currency."
  type        = number
  default     = 5000
}

variable "alert_email" {
  description = "Alert emails. apply.sh passes the signed-in gcloud account; never committed."
  type        = string
  default     = ""
}

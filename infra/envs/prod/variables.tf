variable "project_id" {
  description = "Production project ID. Change it (and the backend bucket in versions.tf) if the ID you created differs."
  type        = string
  default     = "voyagami-prod"
}

variable "region" {
  description = "Single region until VOY-61 decides on data residency."
  type        = string
  default     = "asia-south1"
}

variable "github_repository" {
  type    = string
  default = "msriv/voyagami"
}

variable "billing_account" {
  description = "Billing account ID. apply.sh passes it; empty means no budget."
  type        = string
  default     = ""
}

variable "budget_amount" {
  description = "Monthly budget alert, in the billing account's currency. Set from the Stack Costs plan."
  type        = number
  default     = 10000
}

variable "alert_email" {
  description = "Alert emails. apply.sh passes the signed-in gcloud account; never committed."
  type        = string
  default     = ""
}

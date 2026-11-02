variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "github_repository" {
  description = "Repository allowed to act as the CI account, as owner/name."
  type        = string
}

variable "state_bucket" {
  type = string
}

variable "ci_roles" {
  description = "Project roles for the CI account. Widen only when a ticket needs it."
  type        = list(string)
  default     = ["roles/viewer", "roles/run.developer"]
}

variable "billing_account" {
  description = "Billing account ID (XXXXXX-XXXXXX-XXXXXX). apply.sh reads it from the project."
  type        = string
  default     = ""
}

variable "budget_amount" {
  description = "Monthly budget in the billing account's currency. 0 means no budget alert."
  type        = number
  default     = 0
}

variable "project_id" {
  type = string
}

variable "github_repository" {
  description = "Repository allowed to impersonate the CI account, as owner/name."
  type        = string
}

variable "state_bucket" {
  description = "Terraform state bucket the CI account may read and write."
  type        = string
}

variable "pool_id" {
  type    = string
  default = "github"
}

variable "ci_account_id" {
  type    = string
  default = "ci-deployer"
}

variable "ci_roles" {
  description = "Project roles for the CI account. Later tickets add to this list."
  type        = list(string)
  default     = []
}

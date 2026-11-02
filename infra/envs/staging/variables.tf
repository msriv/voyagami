variable "project_id" {
  description = "Staging shares the dev project."
  type        = string
  default     = "voyagami-dev-512208"
}

variable "region" {
  type    = string
  default = "asia-south1"
}

variable "prefix" {
  description = "Read by apply.sh to find this environment's secrets. Keep in step with main.tf."
  type        = string
  default     = "staging-"
}

variable "ci_service_account_email" {
  description = "CI account created by envs/dev."
  type        = string
  default     = "ci-deployer@voyagami-dev-512208.iam.gserviceaccount.com"
}

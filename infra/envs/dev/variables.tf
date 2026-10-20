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

variable "project_id" {
  description = "GCP project that gets the APIs."
  type        = string
}

variable "apis" {
  description = "Service names to enable, for example run.googleapis.com."
  type        = list(string)
}

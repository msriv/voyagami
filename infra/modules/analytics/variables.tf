variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "prefix" {
  type    = string
  default = ""
}

variable "publisher_email" {
  description = "Service account that publishes events (the API runtime)."
  type        = string
}

variable "raw_retention_days" {
  description = "Raw event partitions are dropped after this many days (14 months). Rollups have no uid and are kept."
  type        = number
  default     = 425
}

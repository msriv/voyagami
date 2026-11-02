variable "project_id" {
  type = string
}

variable "name" {
  description = "Database ID. \"(default)\" for the main one; staging later uses its own name."
  type        = string
  default     = "(default)"
}

variable "location_id" {
  description = "Firestore location, for example asia-south1. Cannot be changed after creation."
  type        = string
}

variable "backups" {
  description = "Point-in-time recovery and a daily backup (14 days)."
  type        = bool
  default     = false
}

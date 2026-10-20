variable "project_id" {
  type = string
}

variable "name" {
  description = "Globally unique bucket name."
  type        = string
}

variable "location" {
  type = string
}

variable "cors_origins" {
  description = "Origins allowed for browser-style signed uploads. Native apps do not send one."
  type        = list(string)
  default     = ["*"]
}

variable "force_destroy" {
  description = "Allow deleting a non-empty bucket. Keep false outside throwaway dev."
  type        = bool
  default     = false
}

variable "project_id" {
  type = string
}

variable "account_id" {
  type = string
}

variable "display_name" {
  type = string
}

variable "roles" {
  description = "Project roles for this account. Keep the list minimal."
  type        = list(string)
  default     = []
}

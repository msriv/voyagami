variable "project_id" {
  type = string
}

variable "topics" {
  description = "Topic names without prefix."
  type        = list(string)
}

variable "name_prefix" {
  description = "Prefix for every topic, for example \"staging-\"."
  type        = string
  default     = ""
}

variable "message_retention" {
  type    = string
  default = "86400s"
}

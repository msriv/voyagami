variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name" {
  type = string
}

variable "service_account_email" {
  description = "Runtime identity of the service."
  type        = string
}

variable "image" {
  description = "Initial image. Later deploys replace it without a Terraform change."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "port" {
  type    = number
  default = 8080
}

variable "min_instances" {
  description = "0 scales to zero."
  type        = number
  default     = 0
}

variable "max_instances" {
  type    = number
  default = 3
}

variable "cpu" {
  type    = string
  default = "1"
}

variable "memory" {
  type    = string
  default = "512Mi"
}

variable "env" {
  description = "Plain environment variables. Secrets use Secret Manager, not this map."
  type        = map(string)
  default     = {}
}

variable "secret_env" {
  description = "Environment variables read from Secret Manager (name to secret ID, latest version). The runtime account needs secretAccessor on each."
  type        = map(string)
  default     = {}
}

variable "allow_unauthenticated" {
  description = "Let anyone reach the URL. The API does its own auth; the mobile app cannot present IAM tokens."
  type        = bool
  default     = false
}

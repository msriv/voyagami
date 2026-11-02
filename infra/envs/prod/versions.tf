terraform {
  required_version = ">= 1.9.0, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }

  # Bucket is created once by infra/scripts/bootstrap-state.sh.
  backend "gcs" {
    bucket = "voyagami-prod-tfstate"
    prefix = "prod"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region

  # The budget API needs a quota project when run with a person's credentials.
  user_project_override = true
  billing_project       = var.project_id
}

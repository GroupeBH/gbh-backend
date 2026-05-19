terraform {
  required_version = ">= 1.5.0"

  backend "gcs" {}

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0, < 7.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

data "google_project" "current" {
  project_id = var.project_id
}

locals {
  default_env_vars = {
    APP_ENV       = "production"
    SERVER_ADDR   = ":${var.container_port}"
    TZ            = "Africa/Kinshasa"
    COOKIE_SECURE = "true"
  }

  env_vars = merge(local.default_env_vars, var.env_vars)

  runtime_service_account = var.service_account_email != "" ? var.service_account_email : "${data.google_project.current.number}-compute@developer.gserviceaccount.com"
}

resource "google_project_service" "required" {
  for_each = var.enable_required_apis ? toset([
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
  ]) : toset([])

  project            = var.project_id
  service            = each.key
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "app" {
  count = var.create_artifact_registry_repository ? 1 : 0

  project       = var.project_id
  location      = var.artifact_registry_location != "" ? var.artifact_registry_location : var.region
  repository_id = var.artifact_registry_repository
  description   = "GBH Backend Docker images"
  format        = "DOCKER"

  depends_on = [google_project_service.required]
}

resource "google_cloud_run_v2_service" "api" {
  name                = var.service_name
  location            = var.region
  project             = var.project_id
  ingress             = var.ingress
  deletion_protection = var.deletion_protection

  template {
    service_account                  = var.service_account_email != "" ? var.service_account_email : null
    max_instance_request_concurrency = var.max_instance_request_concurrency

    scaling {
      min_instance_count = var.min_instance_count
      max_instance_count = var.max_instance_count
    }

    containers {
      image = var.app_image

      ports {
        container_port = var.container_port
      }

      resources {
        limits            = var.resource_limits
        cpu_idle          = true
        startup_cpu_boost = true
      }

      dynamic "env" {
        for_each = local.env_vars

        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = var.secret_env_vars

        content {
          name = env.key

          value_source {
            secret_key_ref {
              secret  = env.value.secret
              version = env.value.version
            }
          }
        }
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  depends_on = [
    google_project_service.required,
    google_secret_manager_secret_iam_member.secret_accessor,
  ]
}

resource "google_secret_manager_secret_iam_member" "secret_accessor" {
  for_each = var.grant_secret_access ? var.secret_env_vars : {}

  project   = var.project_id
  secret_id = each.value.secret
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${local.runtime_service_account}"

  depends_on = [google_project_service.required]
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  count = var.allow_unauthenticated ? 1 : 0

  project  = var.project_id
  location = google_cloud_run_v2_service.api.location
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

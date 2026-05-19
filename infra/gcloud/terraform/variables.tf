variable "project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "region" {
  description = "Cloud Run region."
  type        = string
  default     = "europe-west1"
}

variable "service_name" {
  description = "Cloud Run service name."
  type        = string
  default     = "gbh-backend"
}

variable "app_image" {
  description = "Docker image URI deployed to Cloud Run."
  type        = string
}

variable "artifact_registry_location" {
  description = "Artifact Registry location. Leave empty to use the Cloud Run region."
  type        = string
  default     = ""
}

variable "artifact_registry_repository" {
  description = "Artifact Registry repository ID for Docker images."
  type        = string
  default     = "gbh-backend"
}

variable "create_artifact_registry_repository" {
  description = "Create the Artifact Registry repository with Terraform."
  type        = bool
  default     = true
}

variable "container_port" {
  description = "Container port exposed by the API image."
  type        = number
  default     = 8080
}

variable "min_instance_count" {
  description = "Minimum Cloud Run instances."
  type        = number
  default     = 0
}

variable "max_instance_count" {
  description = "Maximum Cloud Run instances."
  type        = number
  default     = 3
}

variable "max_instance_request_concurrency" {
  description = "Maximum concurrent requests per Cloud Run instance."
  type        = number
  default     = 80
}

variable "resource_limits" {
  description = "Cloud Run container resource limits."
  type        = map(string)
  default = {
    cpu    = "1"
    memory = "512Mi"
  }
}

variable "ingress" {
  description = "Cloud Run ingress setting."
  type        = string
  default     = "INGRESS_TRAFFIC_ALL"
}

variable "allow_unauthenticated" {
  description = "Allow public unauthenticated access to the API."
  type        = bool
  default     = true
}

variable "service_account_email" {
  description = "Optional runtime service account email for Cloud Run."
  type        = string
  default     = ""
}

variable "env_vars" {
  description = "Plain environment variables for Cloud Run."
  type        = map(string)
  default     = {}
}

variable "secret_env_vars" {
  description = "Secret Manager env vars. Keys are env var names."
  type = map(object({
    secret  = string
    version = string
  }))
  default = {}
}

variable "grant_secret_access" {
  description = "Grant the Cloud Run runtime service account access to referenced Secret Manager secrets."
  type        = bool
  default     = true
}

variable "enable_required_apis" {
  description = "Enable required Google Cloud APIs from Terraform."
  type        = bool
  default     = true
}

variable "deletion_protection" {
  description = "Protect the Cloud Run service from Terraform deletion."
  type        = bool
  default     = false
}

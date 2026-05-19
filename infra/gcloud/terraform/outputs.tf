output "service_name" {
  description = "Cloud Run service name."
  value       = google_cloud_run_v2_service.api.name
}

output "service_url" {
  description = "Cloud Run service URL."
  value       = google_cloud_run_v2_service.api.uri
}

output "region" {
  description = "Cloud Run region."
  value       = var.region
}

output "image" {
  description = "Deployed Docker image."
  value       = var.app_image
}

output "artifact_registry_repository" {
  description = "Artifact Registry repository ID."
  value       = var.artifact_registry_repository
}

output "public_ip" {
  description = "Lightsail static public IP address."
  value       = aws_lightsail_static_ip.api.ip_address
}

output "ssh_command" {
  description = "Basic SSH command. Add -i <private-key> if your key is not loaded in ssh-agent."
  value       = "ssh ${var.ssh_user}@${aws_lightsail_static_ip.api.ip_address}"
}

output "app_url" {
  description = "Expected public URL for the API."
  value       = var.domain_name != "" ? "https://${var.domain_name}" : "http://${aws_lightsail_static_ip.api.ip_address}"
}

output "instance_name" {
  description = "Lightsail instance name."
  value       = aws_lightsail_instance.api.name
}

output "region" {
  description = "AWS region."
  value       = var.aws_region
}

output "bundle_id" {
  description = "Lightsail bundle ID."
  value       = var.lightsail_bundle_id
}

output "domain_name" {
  description = "Configured domain name."
  value       = var.domain_name
}

output "ssh_user" {
  description = "SSH user."
  value       = var.ssh_user
}


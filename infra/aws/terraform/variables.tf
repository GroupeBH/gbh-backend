variable "aws_region" {
  description = "AWS region where the Lightsail instance is created."
  type        = string
  default     = "eu-central-1"
}

variable "availability_zone" {
  description = "Lightsail availability zone. Leave empty to use the first AZ for aws_region."
  type        = string
  default     = ""
}

variable "instance_name" {
  description = "Lightsail instance name."
  type        = string
  default     = "gbh-backend-prod"
}

variable "static_ip_name" {
  description = "Lightsail static IP name."
  type        = string
  default     = "gbh-backend-prod-ip"
}

variable "lightsail_blueprint_id" {
  description = "Lightsail blueprint ID. Verify with: aws lightsail get-blueprints --include-inactive."
  type        = string
  default     = "ubuntu_24_04"
}

variable "lightsail_bundle_id" {
  description = "Lightsail bundle ID. micro_3_0 is a small Linux/Unix public IPv4 plan."
  type        = string
  default     = "micro_3_0"
}

variable "key_pair_name" {
  description = "Existing Lightsail key pair name. Leave empty to use the account/region default behavior."
  type        = string
  default     = ""
}

variable "ssh_user" {
  description = "SSH user for the selected blueprint."
  type        = string
  default     = "ubuntu"
}

variable "admin_cidr" {
  description = "CIDR allowed to connect over SSH, for example 203.0.113.10/32."
  type        = string

  validation {
    condition     = can(cidrhost(var.admin_cidr, 0))
    error_message = "admin_cidr must be a valid IPv4 or IPv6 CIDR block."
  }
}

variable "extra_ssh_cidrs" {
  description = "Additional CIDRs allowed to connect over SSH."
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for cidr in var.extra_ssh_cidrs : can(cidrhost(cidr, 0))])
    error_message = "Every value in extra_ssh_cidrs must be a valid IPv4 or IPv6 CIDR block."
  }
}

variable "domain_name" {
  description = "Public API domain used for Terraform outputs and Caddy."
  type        = string
  default     = ""
}

variable "tags" {
  description = "Additional tags for Lightsail resources."
  type        = map(string)
  default = {
    Environment = "production"
  }
}


terraform {
  required_version = ">= 1.5.0"

  backend "s3" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0, < 7.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

locals {
  effective_availability_zone = var.availability_zone != "" ? var.availability_zone : "${var.aws_region}a"
  ssh_cidrs                   = distinct(concat([var.admin_cidr], var.extra_ssh_cidrs))

  common_tags = merge(
    {
      Project   = "gbh-backend"
      ManagedBy = "terraform"
    },
    var.tags
  )
}

resource "aws_lightsail_instance" "api" {
  name              = var.instance_name
  availability_zone = local.effective_availability_zone
  blueprint_id      = var.lightsail_blueprint_id
  bundle_id         = var.lightsail_bundle_id
  key_pair_name     = var.key_pair_name != "" ? var.key_pair_name : null

  tags = local.common_tags
}

resource "aws_lightsail_static_ip" "api" {
  name = var.static_ip_name
}

resource "aws_lightsail_static_ip_attachment" "api" {
  static_ip_name = aws_lightsail_static_ip.api.name
  instance_name  = aws_lightsail_instance.api.name
}

resource "aws_lightsail_instance_public_ports" "api" {
  instance_name = aws_lightsail_instance.api.name

  port_info {
    protocol  = "tcp"
    from_port = 22
    to_port   = 22
    cidrs     = local.ssh_cidrs
  }

  port_info {
    protocol  = "tcp"
    from_port = 80
    to_port   = 80
    cidrs     = ["0.0.0.0/0"]
  }

  port_info {
    protocol  = "tcp"
    from_port = 443
    to_port   = 443
    cidrs     = ["0.0.0.0/0"]
  }
}


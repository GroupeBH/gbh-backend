# AWS Lightsail

This deployment is based on the `uty-infra-lightsail` structure:

- Terraform creates a Lightsail instance, static IP, and public ports.
- Ansible installs Docker, renders Docker Compose and Caddy, then starts the app.
- The app container is pulled from a Docker image repository.

## Local deploy

Required tools: Terraform, Ansible, AWS CLI, SSH.

```bash
cd infra/aws
export AWS_REGION=eu-central-1
export ADMIN_CIDR="$(curl -fsSL https://checkip.amazonaws.com | tr -d '[:space:]')/32"
export APP_IMAGE_REPOSITORY=gbhsarl/gbh-backend
export APP_IMAGE_TAG=latest
export APP_ENV_FILE="$PWD/.env.production"
export DOMAIN_NAME=api.example.com
./deploy.sh
```

## GitHub variables

- `AWS_REGION`
- `ADMIN_CIDR`
- `KEY_PAIR_NAME`
- `LIGHTSAIL_BUNDLE_ID`
- `LIGHTSAIL_BLUEPRINT_ID`
- `INSTANCE_NAME`
- `STATIC_IP_NAME`
- `SSH_USER`
- `DOMAIN_NAME`
- `CADDY_EMAIL`
- `APP_IMAGE_REPOSITORY`
- `APP_IMAGE_TAG`
- `TF_BACKEND_BUCKET`
- `TF_BACKEND_KEY`
- `TF_BACKEND_REGION`

## GitHub secrets

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `LIGHTSAIL_SSH_PRIVATE_KEY`
- `APP_ENV_PRODUCTION`
- `TERRAFORM_TFVARS` optional
- `DOCKERHUB_USERNAME` optional, for building and pushing the image
- `DOCKERHUB_TOKEN` optional, for building and pushing the image


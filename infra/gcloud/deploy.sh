#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${ROOT_DIR}/terraform"
ANSIBLE_DIR="${ROOT_DIR}/ansible"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

set_tf_var_from_env() {
  local env_name="$1"
  local tf_name="$2"
  local value="${!env_name:-}"

  if [[ -n "$value" ]]; then
    export "TF_VAR_${tf_name}=${value}"
  fi
}

require_cmd terraform
require_cmd ansible-playbook

set_tf_var_from_env GCP_PROJECT_ID project_id
set_tf_var_from_env GCP_REGION region
set_tf_var_from_env CLOUD_RUN_SERVICE service_name
set_tf_var_from_env APP_IMAGE app_image

if [[ -z "${TF_VAR_project_id:-}" ]]; then
  echo "Set GCP_PROJECT_ID before deploying." >&2
  exit 1
fi

if [[ -z "${TF_VAR_app_image:-}" ]]; then
  echo "Set APP_IMAGE to the Docker image URI to deploy." >&2
  exit 1
fi

TERRAFORM_INIT_ARGS=()
if [[ "${TERRAFORM_INIT_RECONFIGURE:-0}" == "1" ]]; then
  TERRAFORM_INIT_ARGS+=("-reconfigure")
fi

echo "Running Terraform init..."
if [[ -f "${TERRAFORM_DIR}/backend.hcl" ]]; then
  terraform -chdir="$TERRAFORM_DIR" init "${TERRAFORM_INIT_ARGS[@]}" -backend-config=backend.hcl
else
  terraform -chdir="$TERRAFORM_DIR" init "${TERRAFORM_INIT_ARGS[@]}" -backend=false
fi

echo "Applying Terraform..."
terraform -chdir="$TERRAFORM_DIR" apply -auto-approve

SERVICE_URL="$(terraform -chdir="$TERRAFORM_DIR" output -raw service_url)"

echo "Verifying Cloud Run service..."
ansible-playbook \
  -i "${ANSIBLE_DIR}/inventory.ini.example" \
  "${ANSIBLE_DIR}/playbook.yml" \
  -e "service_url=${SERVICE_URL}"

echo
echo "Deployment complete."
echo "Service URL: ${SERVICE_URL}"

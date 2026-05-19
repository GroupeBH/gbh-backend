#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="${ROOT_DIR}/terraform"
ANSIBLE_DIR="${ROOT_DIR}/ansible"
INVENTORY_FILE="${ANSIBLE_DIR}/inventory.ini"

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
require_cmd aws
require_cmd ssh

APP_IMAGE_REPOSITORY="${APP_IMAGE_REPOSITORY:-gbhsarl/gbh-backend}"
APP_IMAGE_TAG="${APP_IMAGE_TAG:-latest}"
APP_ENV_FILE="${APP_ENV_FILE:-${ROOT_DIR}/.env.production}"
CADDY_EMAIL="${CADDY_EMAIL:-}"
SSH_PRIVATE_KEY_PATH="${SSH_PRIVATE_KEY_PATH:-}"

DOMAIN_NAME_ENV_IS_SET=0
if [[ "${DOMAIN_NAME+x}" == "x" ]]; then
  DOMAIN_NAME_ENV_IS_SET=1
else
  DOMAIN_NAME=""
fi

if [[ ! -f "$APP_ENV_FILE" ]]; then
  echo "Application env file not found: $APP_ENV_FILE" >&2
  echo "Set APP_ENV_FILE or create .env.production in infra/aws." >&2
  exit 1
fi

set_tf_var_from_env AWS_REGION aws_region
set_tf_var_from_env ADMIN_CIDR admin_cidr
set_tf_var_from_env KEY_PAIR_NAME key_pair_name
set_tf_var_from_env LIGHTSAIL_BUNDLE_ID lightsail_bundle_id
set_tf_var_from_env LIGHTSAIL_BLUEPRINT_ID lightsail_blueprint_id
set_tf_var_from_env INSTANCE_NAME instance_name
set_tf_var_from_env STATIC_IP_NAME static_ip_name
set_tf_var_from_env SSH_USER ssh_user

if [[ "$DOMAIN_NAME_ENV_IS_SET" -eq 1 ]]; then
  export TF_VAR_domain_name="$DOMAIN_NAME"
fi

echo "Checking AWS credentials..."
aws sts get-caller-identity >/dev/null

LIGHTSAIL_BUNDLE_ID_EFFECTIVE="${TF_VAR_lightsail_bundle_id:-}"
if [[ -z "$LIGHTSAIL_BUNDLE_ID_EFFECTIVE" && -f "${TERRAFORM_DIR}/terraform.tfvars" ]]; then
  LIGHTSAIL_BUNDLE_ID_EFFECTIVE="$(
    sed -n 's/^[[:space:]]*lightsail_bundle_id[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' "${TERRAFORM_DIR}/terraform.tfvars" | tail -n 1
  )"
fi
LIGHTSAIL_BUNDLE_ID_EFFECTIVE="${LIGHTSAIL_BUNDLE_ID_EFFECTIVE:-micro_3_0}"
AWS_REGION_EFFECTIVE="${AWS_REGION:-${AWS_DEFAULT_REGION:-eu-central-1}}"

echo "Checking Lightsail bundle availability for ${LIGHTSAIL_BUNDLE_ID_EFFECTIVE} in ${AWS_REGION_EFFECTIVE}..."
available_bundle_id="$(
  aws lightsail get-bundles \
    --query "bundles[?bundleId=='${LIGHTSAIL_BUNDLE_ID_EFFECTIVE}'].bundleId | [0]" \
    --output text
)"
if [[ "$available_bundle_id" != "$LIGHTSAIL_BUNDLE_ID_EFFECTIVE" ]]; then
  echo "Lightsail bundle '${LIGHTSAIL_BUNDLE_ID_EFFECTIVE}' is not available in region '${AWS_REGION_EFFECTIVE}'." >&2
  echo "Update LIGHTSAIL_BUNDLE_ID or TERRAFORM_TFVARS with a valid bundle." >&2
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

PUBLIC_IP="$(terraform -chdir="$TERRAFORM_DIR" output -raw public_ip)"
INSTANCE_NAME_OUT="$(terraform -chdir="$TERRAFORM_DIR" output -raw instance_name)"
REGION_OUT="$(terraform -chdir="$TERRAFORM_DIR" output -raw region)"
BUNDLE_ID_OUT="$(terraform -chdir="$TERRAFORM_DIR" output -raw bundle_id)"
SSH_USER_OUT="$(terraform -chdir="$TERRAFORM_DIR" output -raw ssh_user)"
DOMAIN_FROM_TF="$(terraform -chdir="$TERRAFORM_DIR" output -raw domain_name)"

if [[ "$DOMAIN_NAME_ENV_IS_SET" -eq 0 && -z "$DOMAIN_NAME" ]]; then
  DOMAIN_NAME="$DOMAIN_FROM_TF"
fi

if [[ -z "${SSH_USER:-}" ]]; then
  SSH_USER="$SSH_USER_OUT"
fi

APP_ENV_FILE_ABS="$(cd "$(dirname "$APP_ENV_FILE")" && pwd)/$(basename "$APP_ENV_FILE")"

echo "Generating Ansible inventory for ${INSTANCE_NAME_OUT} (${PUBLIC_IP})..."
{
  echo "[gbh_backend]"
  echo "${INSTANCE_NAME_OUT} ansible_host=${PUBLIC_IP}"
  echo
  echo "[gbh_backend:vars]"
  echo "ansible_user=${SSH_USER}"
  echo "ansible_python_interpreter=/usr/bin/python3"
  if [[ -n "$SSH_PRIVATE_KEY_PATH" ]]; then
    echo "ansible_ssh_private_key_file=${SSH_PRIVATE_KEY_PATH}"
  fi
} >"$INVENTORY_FILE"
chmod 0600 "$INVENTORY_FILE"

SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o ConnectTimeout=5)
if [[ -n "$SSH_PRIVATE_KEY_PATH" ]]; then
  SSH_OPTS+=(-i "$SSH_PRIVATE_KEY_PATH")
fi

echo "Waiting for SSH on ${PUBLIC_IP}..."
for attempt in $(seq 1 60); do
  if ssh "${SSH_OPTS[@]}" "${SSH_USER}@${PUBLIC_IP}" "true" >/dev/null 2>&1; then
    break
  fi

  if [[ "$attempt" -eq 60 ]]; then
    echo "SSH did not become available after 5 minutes." >&2
    exit 1
  fi

  sleep 5
done

echo "Deploying with Ansible..."
ansible-playbook \
  -i "$INVENTORY_FILE" \
  "${ANSIBLE_DIR}/playbook.yml" \
  -e "app_image_repository=${APP_IMAGE_REPOSITORY}" \
  -e "app_image_tag=${APP_IMAGE_TAG}" \
  -e "app_env_file=${APP_ENV_FILE_ABS}" \
  -e "domain_name=${DOMAIN_NAME}" \
  -e "caddy_email=${CADDY_EMAIL}"

APP_URL="$(terraform -chdir="$TERRAFORM_DIR" output -raw app_url)"

echo
echo "Deployment complete."
echo "Region: ${REGION_OUT}"
echo "Bundle: ${BUNDLE_ID_OUT}"
echo "Public IP: ${PUBLIC_IP}"
echo "App URL: ${APP_URL}"

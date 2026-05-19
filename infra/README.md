# GBH Backend Infrastructure

This directory is split by cloud provider:

- `gcloud`: Google Cloud Run deployment with Terraform, plus an Ansible endpoint check.
- `aws`: AWS Lightsail deployment with Terraform and Ansible.
- `alibabacloud`: placeholder for a future Alibaba Cloud deployment.

Each deploy uses a Docker image built from this project. CI workflows live in
`.github/workflows`:

- `gcloud.yml`
- `aws-lightsail.yml`
- `alibabacloud.yml`

## Required runtime configuration

The application needs production environment variables such as `MONGO_URI`,
`JWT_SECRET`, `ADMIN_API_KEY`, and optional Redis/Brevo/Firebase variables.

For Lightsail, put the full production env file in the GitHub secret
`APP_ENV_PRODUCTION`. The workflow writes it to `infra/aws/.env.production`
before Ansible copies it to the server.

For Cloud Run, use repository secrets:

- `CLOUD_RUN_ENV_VARS_JSON`: optional JSON object of plain env vars.
- `CLOUD_RUN_SECRET_ENV_VARS_JSON`: optional JSON object of Secret Manager refs.

Example `CLOUD_RUN_SECRET_ENV_VARS_JSON`:

```json
{
  "MONGO_URI": { "secret": "gbh-mongo-uri", "version": "latest" },
  "JWT_SECRET": { "secret": "gbh-jwt-secret", "version": "latest" }
}
```

## Terraform state

CI expects remote Terraform state:

- AWS: set repository variables `TF_BACKEND_BUCKET`, optionally
  `TF_BACKEND_KEY` and `TF_BACKEND_REGION`.
- Google Cloud: set repository variable `GCP_TF_STATE_BUCKET`, optionally
  `GCP_TF_STATE_PREFIX`.


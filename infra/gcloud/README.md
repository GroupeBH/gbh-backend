# Google Cloud Run

Terraform deploys the GBH backend image to Cloud Run. The GitHub Actions
workflow builds this project Docker image, pushes it to Artifact Registry, then
applies Terraform.

Artifact Registry is declared in Terraform. The workflow runs a targeted
Terraform apply first so the Docker repository exists before the image push.

Ansible is used here as a lightweight post-deploy check against the Cloud Run
URL because Cloud Run itself is managed by Terraform.

## GitHub variables

- `GCP_PROJECT_ID`
- `GCP_REGION` default `europe-west1`
- `GAR_LOCATION` default same as `GCP_REGION`
- `GAR_REPOSITORY` default `gbh-backend`
- `CLOUD_RUN_SERVICE` default `gbh-backend`
- `CREATE_ARTIFACT_REGISTRY_REPOSITORY` default `true`; set `false` if the
  repository already exists and is not managed by this Terraform state.
- `APP_IMAGE_TAG`
- `GCP_TF_STATE_BUCKET`
- `GCP_TF_STATE_PREFIX`

## GitHub secrets

- `GCP_SERVICE_ACCOUNT_KEY`: JSON key for a service account allowed to push to
  Artifact Registry, manage Cloud Run, and access the Terraform state bucket.
- `CLOUD_RUN_ENV_VARS_JSON`: optional JSON object of plain environment vars.
- `CLOUD_RUN_SECRET_ENV_VARS_JSON`: optional JSON object mapping env var names
  to Secret Manager refs.

When `CLOUD_RUN_SECRET_ENV_VARS_JSON` is set, Terraform grants the Cloud Run
runtime service account `roles/secretmanager.secretAccessor` on those secrets by
default.

Example plain env vars:

```json
{
  "APP_ENV": "production",
  "SERVER_ADDR": ":8080",
  "FRONTEND_ORIGINS": "https://www.gbh.sarl"
}
```

Example secret env vars:

```json
{
  "MONGO_URI": { "secret": "gbh-mongo-uri", "version": "latest" },
  "JWT_SECRET": { "secret": "gbh-jwt-secret", "version": "latest" }
}
```

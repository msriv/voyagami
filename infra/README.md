# infra

Terraform, one root per environment under `envs/`. Only `dev` exists in the D phases (VOY-8). Reusable pieces live in `modules/`.

Projects: `voyagami-dev-512208` (the display name is voyagami-dev) holds dev now (staging joins it in P1). `voyagami-prod` is separate and not created yet.

```
envs/dev/              root for the dev project (remote state in gs://voyagami-dev-512208-tfstate)
modules/project-apis/  enables Google APIs
modules/github-wif/    workload identity pool + CI service account for GitHub Actions
scripts/               one-time bootstrap helpers
```

## First-time setup (once, by a human, in Cloud Shell)

Claude Code web cannot hold Google credentials, and the repo forbids long-lived keys, so the first `apply` runs in [Cloud Shell](https://shell.cloud.google.com) signed in as the project Owner. After that, CI applies through workload identity and no one needs a key.

```sh
git clone https://github.com/msriv/voyagami && cd voyagami
gcloud config set project voyagami-dev-512208
infra/scripts/bootstrap-state.sh               # creates the state bucket
cd infra/envs/dev
terraform init
terraform plan
terraform apply
```

`apply` enables the APIs and creates the workload identity pool, the `ci-deployer` service account and its GitHub binding. It prints two outputs for GitHub Actions (`google-github-actions/auth`):

- `workload_identity_provider`
- `ci_service_account_email`

Store them as GitHub repository variables `GCP_WORKLOAD_IDENTITY_PROVIDER` and `GCP_CI_SERVICE_ACCOUNT` (Settings, Secrets and variables, Actions, Variables). They are not sensitive.

## Rules

- Never commit state, `.tfvars` or keys.
- `ci-deployer` starts with `roles/viewer`. Each ticket that needs more (Cloud Run, Firestore, Pub/Sub) adds only its own roles in `envs/dev/main.tf`.
- Region defaults to `asia-south1`. Change it in `envs/dev/variables.tf` before the first `apply` if you want another.

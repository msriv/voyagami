# infra

Terraform, one root per environment under `envs/`. Only `dev` exists in the D phases (VOY-8). Reusable pieces live in `modules/`.

Projects: `voyagami-dev` holds dev now (staging joins it in P1). `voyagami-prod` is separate and not created yet.

```
envs/dev/              root for voyagami-dev (remote state in gs://voyagami-dev-tfstate)
modules/project-apis/  enables Google APIs
modules/github-wif/    workload identity pool + CI service account for GitHub Actions
scripts/               one-time bootstrap helpers
```

## First-time setup (once, by a human)

Needs an account with Owner on `voyagami-dev` and billing linked.

```sh
gcloud auth login
gcloud auth application-default login
infra/scripts/bootstrap-state.sh voyagami-dev asia-south1   # creates the state bucket
cd infra/envs/dev
terraform init
terraform plan
terraform apply
```

`apply` enables the APIs and creates the workload identity pool, the `ci-deployer` service account and its GitHub binding. It prints two outputs for GitHub Actions (`google-github-actions/auth`):

- `workload_identity_provider`
- `ci_service_account_email`

Store them as GitHub repository variables (not secrets; they are not sensitive). No service account keys are ever created.

## Rules

- Never commit state, `.tfvars` or keys.
- `ci-deployer` starts with `roles/viewer`. Each ticket that needs more (Cloud Run, Firestore, Pub/Sub) adds only its own roles in `envs/dev/main.tf`.
- Region defaults to `asia-south1`. Change it in `envs/dev/variables.tf` before the first `apply` if you want another.

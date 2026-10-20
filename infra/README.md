# infra

Terraform, one root per environment under `envs/`. Only `dev` exists in the D phases (VOY-8). Reusable pieces live in `modules/`.

## Projects

Two Google Cloud projects (decided 2026-10-20):

| Project               | Holds                                                                                                                                                   |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `voyagami-dev-512208` | dev now; staging is added here in P1 (VOY-81) as its own Cloud Run service, named Firestore database and service accounts, with a `staging` name prefix |
| `voyagami-prod`       | production only, created in P1 (VOY-81)                                                                                                                 |

Prod has its own project because Firebase Auth users, push (FCM), IAM, quotas and budgets are all per project. There is no `voyagami-staging` project. The dev project's display name is `voyagami-dev`; its ID is the one above.

## Layout

```
envs/dev/              root for the dev project (remote state in gs://voyagami-dev-512208-tfstate)
modules/project-apis/  enables Google APIs
modules/github-wif/    workload identity pool + CI service account for GitHub Actions
modules/artifact-registry/  Docker image repository
modules/service-account/    service account with a minimal role list
modules/firestore/         Firestore database (location cannot change later)
modules/pubsub-topics/     Pub/Sub topics
modules/storage-bucket/    private bucket with CORS for signed URLs
modules/secrets/           empty Secret Manager secrets (values added by hand)
modules/cloud-run/     Cloud Run service (scale to zero, image owned by deploys)
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

## Applying later changes

Run `terraform plan` then `terraform apply` in `infra/envs/dev` from Cloud Shell, as for the first apply. The first Cloud Run service starts on Google's placeholder image; deploying the API replaces the image and Terraform ignores that change.

## Deploying the API (dev)

`make deploy-dev` builds `services/api/Dockerfile` for linux/amd64, pushes it to Artifact Registry (`voyagami`) tagged with the git commit, rolls `voyagami-api` to it and checks `/health`. It needs `docker` and a signed-in `gcloud` (`gcloud auth login`, project `voyagami-dev-512208`). The same script runs from GitHub: Actions, "Deploy API (dev)", Run workflow, on `main` only. It signs in through workload identity, so no key is stored.

## Rules

- Never commit state, `.tfvars` or keys.
- `ci-deployer` starts with `roles/viewer`. Each ticket that needs more (Cloud Run, Firestore, Pub/Sub) adds only its own roles in `envs/dev/main.tf`.
- Region defaults to `asia-south1`. Change it in `envs/dev/variables.tf` before the first `apply` if you want another.

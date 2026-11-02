# infra

Terraform, one root per environment under `envs/` (`dev`, `staging`, `prod`). Reusable pieces live in `modules/`.

## Projects

Two Google Cloud projects (decided 2026-10-20):

| Project               | Holds                                                                                                                                                   |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `voyagami-dev-512208` | dev now; staging is added here in P1 (VOY-81) as its own Cloud Run service, named Firestore database and service accounts, with a `staging` name prefix |
| `voyagami-prod`       | production only, created in P1 (VOY-81)                                                                                                                 |

Prod has its own project because Firebase Auth users, push (FCM), IAM, quotas and budgets are all per project. There is no `voyagami-staging` project. The dev project's display name is `voyagami-dev`; its ID is the one above.

## Layout

```
envs/dev/                    dev (state in gs://voyagami-dev-512208-tfstate, prefix dev)
envs/staging/                staging inside the dev project, names prefixed staging- (same bucket, prefix staging)
envs/prod/                   production project (state in gs://voyagami-prod-tfstate)
modules/project-base/        one per GCP project: APIs, CI identity (workload identity), Artifact Registry, budget alert
modules/app-stack/           one running copy of the app: Cloud Run API, runtime account, Firestore, Pub/Sub, bucket, secrets, scheduler jobs
modules/<small modules>/     building blocks used by the two above
scripts/apply.sh             the one command that sets up or updates an environment
scripts/bootstrap-state.sh   state bucket (called by apply.sh)
scripts/deploy-api-dev.sh    builds and deploys the API image
```

Every name in `app-stack` takes a prefix, so staging can live inside `voyagami-dev` with a `staging-` prefix.

## Setting up or updating an environment (one command)

Claude Code web cannot hold Google credentials, and the repo forbids long-lived keys, so `apply` runs in [Cloud Shell](https://shell.cloud.google.com) signed in as the project Owner:

```sh
git clone https://github.com/msriv/voyagami && cd voyagami    # or: git pull
infra/scripts/apply.sh dev
```

`apply.sh` is safe to re-run. It:

1. creates the state bucket if it is missing;
2. enables the APIs and creates the empty secrets;
3. gives `auth-secret`, `jobs-secret` and `revenuecat-webhook-secret` a random value if they have none (values go straight to Secret Manager, never through Terraform);
4. stops if `google-maps-server-key` is empty and prints how to add it (an API key only a person can create);
5. runs `terraform apply` for everything else (you confirm with `yes`).

The budget alert is created on the project's own billing account, which the script looks up. If you made a budget by hand before, delete it in Billing, Budgets so you don't get two emails.

After the first apply, store two outputs as GitHub repository variables (Settings, Secrets and variables, Actions, Variables). They are not sensitive:

- `workload_identity_provider` as `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `ci_service_account_email` as `GCP_CI_SERVICE_ACCOUNT`

The first Cloud Run service starts on Google's placeholder image; deploying the API replaces the image and Terraform ignores that change.

### Order and what each environment needs

1. `infra/scripts/apply.sh dev` (staging reuses dev's APIs, registry and CI identity).
2. `infra/scripts/apply.sh staging`. It has its own Firestore database (`staging`), so the API reads `FIRESTORE_DATABASE`. Its Maps key secret is `staging-google-maps-server-key`; the dev key can be reused.
3. Production, once:

   ```sh
   gcloud projects create voyagami-prod --name voyagami-prod     # take another ID if it is taken
   gcloud billing projects link voyagami-prod --billing-account <ID from: gcloud billing accounts list>
   ```

   If the ID differs, change `project_id` in `envs/prod/variables.tf` and the bucket in `envs/prod/versions.tf`. Then add Firebase to the project in the Firebase console (Phone sign-in on) and run `infra/scripts/apply.sh prod`. Prod runs one warm API instance (`min_instances = 1`). The prod CI outputs go into GitHub variables for the prod deploy job (VOY-14).

### Moving dev from the old flat root (once)

VOY-225 moved dev into the two modules. `moved` blocks in `envs/dev/main.tf` keep every live resource; the first plan should show moves, a few in-place changes (`APP_ENV`, `PLACES_API_KEY`, `JOBS_SECRET` and `REVENUECAT_WEBHOOK_SECRET` on the service, the registry description) and new resources (budget, scheduler jobs), and **no destroys**. If it shows a destroy, stop and say so.

If you created the `trial-reminders` or `account-deletions` scheduler jobs by hand, delete them first (`gcloud scheduler jobs delete trial-reminders --location=asia-south1`), or Terraform fails with "already exists".

## Secrets

All values live in Secret Manager. Cloud Run reads `latest` when a revision starts.

| Secret                      | API variable                            | Filled by                                    |
| --------------------------- | --------------------------------------- | -------------------------------------------- |
| `auth-secret`               | `AUTH_SECRET` (signs sign-in tokens)    | `apply.sh`, random                           |
| `jobs-secret`               | `JOBS_SECRET` (bearer for `/v1/jobs/*`) | `apply.sh`, random                           |
| `revenuecat-webhook-secret` | `REVENUECAT_WEBHOOK_SECRET`             | `apply.sh`, random                           |
| `google-maps-server-key`    | `PLACES_API_KEY`                        | you (API key restricted to Places API (New)) |

To rotate one: `openssl rand -hex 32 | tr -d '\n' | gcloud secrets versions add <secret> --data-file=-`, then deploy the API again (and re-run `apply.sh` for `jobs-secret`, so the scheduler gets the new value). Rotating `auth-secret` signs everyone out.

The scheduler jobs read `jobs-secret` through Terraform, so that one value is also in the (private, versioned) state bucket.

## Monitoring

Each environment gets (in `modules/app-stack/monitoring.tf`): an uptime check on `/health` every 5 minutes, alerts for "API down" (10 minutes), more than 5 server errors in 5 minutes, and p95 latency over 2 s for 10 minutes, plus a dashboard named after the service (requests by status, p95 latency, instances, CPU). Alerts email the account that ran `apply.sh`; set `ALERT_EMAIL=...` before running it to use another address. Google sends a confirmation email the first time.

## Backups (prod)

`backups = true` in `envs/prod`: Firestore point-in-time recovery (7 days) and a daily backup kept 14 days, and versioning on the media bucket (old versions removed after 30 days). Dev and staging have none. Restore steps live in the VOY-74 runbook.

## Booking import (Vertex AI)

The booking import endpoint calls Gemini on Vertex with the API's own service account, so there is no key. `terraform apply` grants `roles/aiplatform.user` and sets `VERTEX_LOCATION` (`us-central1`; Gemini may not be offered in `asia-south1`). Pasted booking text and PDFs are sent to that region. Without the apply, the endpoint answers 503. Redeploy the API after the apply.

## RevenueCat webhook

RevenueCat tells the API about purchases at `POST /v1/billing/revenuecat` with a shared secret in the `Authorization` header. `apply.sh` prints the value to paste (`Bearer <revenuecat-webhook-secret>`). In RevenueCat (sandbox project from VOY-85), point the webhook at `<api url>/v1/billing/revenuecat` and paste that value. The app must set the RevenueCat user ID to the account ID, and set a `tripId` attribute before a Trip Pass purchase.

## Scheduled jobs

Terraform creates the Cloud Scheduler jobs (in `modules/app-stack`), each calling the API with `Authorization: Bearer <jobs-secret>`. Running a job twice is harmless.

| Job                 | Schedule (UTC) | Does                                             |
| ------------------- | -------------- | ------------------------------------------------ |
| `trial-reminders`   | every 3 hours  | "your trial ends in 2 days" push                 |
| `account-deletions` | 03:00 daily    | deletes accounts whose 30 days are over (VOY-72) |

`/v1/jobs/recap-videos` gets its job when a renderer exists (docs/recap-video.md).

## Deploying the API (dev)

`make deploy-dev` builds `services/api/Dockerfile` for linux/amd64, pushes it to Artifact Registry (`voyagami`) tagged with the git commit, rolls `voyagami-api` to it and checks `/health`. It needs `docker` and a signed-in `gcloud` (`gcloud auth login`, project `voyagami-dev-512208`). The same script runs from GitHub: Actions, "Deploy API (dev)", Run workflow, on `main` only. It signs in through workload identity, so no key is stored.

## Rules

- Never commit state, `.tfvars` or keys.
- `ci-deployer` starts with `roles/viewer` and `roles/run.developer` (`ci_roles` in `modules/project-base`). Widen only when a ticket needs it.
- Anything that can be Terraform is Terraform. What cannot (creating a project, API keys, store and partner accounts) is a `needs-human` ticket.
- Region defaults to `asia-south1`. Change it in `envs/<env>/variables.tf` before the first `apply` if you want another.

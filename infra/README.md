# infra

Terraform, one root per environment. Only `dev` exists in the D phases (VOY-8).

## Projects

Two Google Cloud projects (decided 2026-10-20):

| Project | Holds |
|---|---|
| `voyagami-dev` | dev now; staging is added here in P1 (VOY-81) as its own Cloud Run service, named Firestore database and service accounts, with a `staging` name prefix |
| `voyagami-prod` | production only, created in P1 (VOY-81) |

Prod has its own project because Firebase Auth users, push (FCM), IAM, quotas and budgets are all per project. There is no `voyagami-staging` project.

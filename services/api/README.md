# services/api

The Go API that runs on Cloud Run.

```
cmd/api/              main: config, logger, server, graceful shutdown
internal/api/         server interface generated from api/openapi.yaml (do not edit)
internal/config/      environment settings and validation
internal/logging/     JSON logs in the Cloud Logging shape
internal/auth/        Verifier interface and Identity (the real verifier comes with sign-in)
internal/server/      router, auth, request IDs, error shape, panic recovery
internal/model/       plain types and validation for stored data (see docs/data-model.md)
internal/repo/        storage interfaces the rest of the API depends on
internal/repo/firestorerepo/  Firestore implementation of those interfaces
```

## The API contract

`api/openapi.yaml` (OpenAPI 3.1, at the repo root) is the source of truth. The routes, request and response types in `internal/api/api.gen.go` are generated from it. After changing the spec, run `make api-client` and commit the result. CI fails if the generated code is out of date. Handlers live in `internal/server` and implement the generated `StrictServerInterface`.

The generator is pinned as a Go tool in `go.mod` (`go tool oapi-codegen`), so nothing is installed globally. Go 1.25 is required because that generator version needs it.

## Run it

```sh
make dev            # from the repo root; listens on :8080
curl localhost:8080/healthz
```

## Settings

All optional. Bad values stop the process at startup with every problem listed.

| Variable           | Default | Meaning                                        |
| ------------------ | ------- | ---------------------------------------------- |
| `APP_ENV`          | `dev`   | `dev`, `staging` or `prod`                     |
| `PORT`             | `8080`  | Listen port (Cloud Run sets this)              |
| `LOG_LEVEL`        | `info`  | `debug`, `info`, `warn` or `error`             |
| `SHUTDOWN_TIMEOUT` | `10s`   | Time allowed for in-flight requests on SIGTERM |
| `REQUEST_TIMEOUT`  | `30s`   | Maximum time for one request                   |

## Container and deploy

`Dockerfile` builds a static binary into a distroless, non-root image that listens on `PORT` (Cloud Run sets it). `make deploy-dev` ships it to the dev Cloud Run service; see `infra/README.md`. CI builds the image on every API change but does not push it.

## Data access

Code outside `internal/repo/firestorerepo` uses the `repo.Users` and `repo.Trips` interfaces and the types in `internal/model`, never the Firestore client, so the planned Cloud SQL move only replaces one package. Missing and duplicate documents come back as `repo.ErrNotFound` and `repo.ErrAlreadyExists`, rule breaks (such as removing a trip's owner) as `repo.ErrForbidden`, and bad input as `*model.ValidationError`. Membership changes keep the trip's `memberUids` list in step inside one transaction. Checks here are about shape only (a currency code has 3 capital letters); whether it is a real ISO 4217 code is the locale module's job (VOY-58).

The repository tests run against the Firestore emulator. `make test` starts it for you (needs Java 21; the first run downloads the emulator). Plain `go test ./...` skips those tests unless `FIRESTORE_EMULATOR_HOST` is set, and fails instead of skipping when `REQUIRE_EMULATOR=1`, which CI sets. Each test makes its own IDs, so tests never need a clean database.

## Auth, errors and request IDs

Every route needs `Authorization: Bearer <token>` unless the spec marks it `security: []` (a test keeps `publicRoutes` in `internal/server/auth.go` in step with the spec). The token is checked by an `auth.Verifier`; until sign-in exists `main` uses `auth.RejectAll`, so protected routes answer 401. Tests use a fake verifier.

All errors use the `Error` schema (`code`, `message`, `requestId`), including router 404 and 405 and recovered panics (500). Every response carries `X-Request-Id`; a safe caller-supplied value is reused. Logs carry the request ID, never the token.

## Logging rule

Logs carry no personal data: no phone numbers, exact locations or free text. The request log records method, path (without the query string), status, duration and request ID only. Never log request or response bodies.

Lint config is `.golangci.yml` at the repo root. `make lint test` runs golangci-lint and `go test` here.

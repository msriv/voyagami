# services/api

The Go API that runs on Cloud Run. Standard library only for now.

```
cmd/api/              main: config, logger, server, graceful shutdown
internal/api/         server interface generated from api/openapi.yaml (do not edit)
internal/config/      environment settings and validation
internal/logging/     JSON logs in the Cloud Logging shape
internal/server/      router and request logging
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

## Logging rule

Logs carry no personal data: no phone numbers, exact locations or free text. The request log records method, path (without the query string), status and duration only. Never log request or response bodies.

Lint config is `.golangci.yml` at the repo root. `make lint test` runs golangci-lint and `go test` here.

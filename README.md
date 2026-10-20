# Voyagami

A mobile-only group travel planner and journal for iOS and Android. There is no website.

- Work is tracked in Linear, team **VOY**: https://linear.app/voyagami
- Rules for contributors (human or Claude) are in [CLAUDE.md](CLAUDE.md). Read it first.
- Design boards are in [`design/`](design/BOARD_INDEX.md). They are reference, not code to copy.
- Engineering docs are in [`docs/`](docs/README.md).

## Set up a fresh laptop

These steps assume macOS or Linux. On Windows, use WSL 2 for everything except the Android emulator.

### 1. Install the toolchain

| Tool                                                                                        | Version                                | Why                                                    |
| ------------------------------------------------------------------------------------------- | -------------------------------------- | ------------------------------------------------------ |
| [Git](https://git-scm.com/)                                                                 | any recent                             | source control                                         |
| [Node.js](https://nodejs.org/) via [nvm](https://github.com/nvm-sh/nvm) or similar          | 22 (see [`.nvmrc`](.nvmrc))            | app, shared packages, lint                             |
| pnpm, via [Corepack](https://nodejs.org/api/corepack.html)                                  | pinned in `package.json`               | package manager; `make setup` enables Corepack for you |
| [Go](https://go.dev/dl/)                                                                    | 1.24 or newer                          | the API in `services/api`                              |
| [golangci-lint](https://golangci-lint.run/welcome/install/)                                 | v2                                     | Go lint (config in `.golangci.yml`)                    |
| [Xcode](https://developer.apple.com/xcode/) (macOS only)                                    | latest stable, with an iOS simulator   | iOS builds and the simulator                           |
| [Android Studio](https://developer.android.com/studio)                                      | latest stable, with one emulator image | Android builds and the emulator                        |
| [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) and the Firebase emulator kit | latest                                 | dev GCP project and local emulators (from VOY-8)       |

On macOS with Homebrew:

```sh
brew install git nvm go golangci-lint
brew install --cask google-cloud-sdk android-studio
nvm install   # reads .nvmrc
```

### 2. Clone and install

```sh
git clone https://github.com/msriv/voyagami.git
cd voyagami
nvm use
make setup
```

`make setup` enables Corepack, installs JavaScript dependencies from the lockfile, and downloads Go modules once the API exists.

### 3. Check it works

```sh
make lint test
```

Both must pass before you open a PR.

### 4. Run it

```sh
make dev
```

This is a placeholder until the API (VOY-91) and the app (VOY-98) exist. When they do, it starts the API against the dev emulators and the Expo dev server. The dev setup never needs real accounts: sign-in uses a stub SMS provider whose code is always `123456`.

Never commit secrets. Local settings go in `.env` files, which git ignores; commit an `.env.example` with placeholder values instead.

## Commands

| Command           | What it does                                                 |
| ----------------- | ------------------------------------------------------------ |
| `make setup`      | Install toolchain dependencies                               |
| `make dev`        | Run the API and the app against dev                          |
| `make lint`       | Prettier check, ESLint, typecheck and Go lint                |
| `make test`       | Run all tests                                                |
| `make format`     | Rewrite files with Prettier and gofmt                        |
| `make api-client` | Generate the typed API client from the OpenAPI spec (VOY-16) |

## Repo layout

```
apps/mobile      Expo (React Native) app
services/api     Go API on Cloud Run
packages/        shared types, generated api-client
infra/           Terraform (one root per environment)
design/          boards, photos, tokens (read-only reference)
docs/            engineering docs
```

## Branches and pull requests

- One Linear ticket per branch and per PR. Keep PRs small.
- Name the branch after the ticket, using the branch name Linear suggests (the "Copy git branch name" action), for example `voy-87`. Add a short slug if you like: `voy-87-docs-skeleton`. The `VOY-123` key in the branch name or PR title links the PR to the ticket.
- Title the PR `VOY-123: <ticket title>`. The [PR template](.github/pull_request_template.md) asks for a short "How I tested this" and screenshots for any UI change.
- A human reviews, merges and closes the ticket.

## Continuous integration

GitHub Actions ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs on every pull request and on pushes to `main`. Path filters decide which parts run:

| Job                             | Runs when these change                      | What it checks                             |
| ------------------------------- | ------------------------------------------- | ------------------------------------------ |
| Mobile and shared packages      | `apps/`, `packages/`, JS config, docs       | Prettier, ESLint, typecheck, unit tests    |
| Go API                          | `services/api/`, `.golangci.yml`            | gofmt, `go vet`, golangci-lint, `go test`  |
| Firestore rules on the emulator | `firebase.json`, `firestore.rules`, the API | Rules tests against the Firestore emulator |

The Go and rules jobs skip themselves until `services/api/go.mod` (VOY-91) and the Firestore rules (VOY-97) exist. Dependencies are cached, and each job should finish in under 5 minutes.

### Required status checks

Set one required check in the GitHub branch protection rule for `main`: **CI**. It is the final job in the workflow, runs every time, and fails if any job it depends on failed or was cancelled, so skipped (path-filtered) jobs never leave the check pending.

Deploys, EAS builds and security scanning are not part of this workflow; they arrive in P1.

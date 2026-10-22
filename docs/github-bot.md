# GitHub bot for Claude's pull requests

Claude Code sessions push and call GitHub as Mihir (`msriv`), so a PR that Claude opens directly is Mihir's own PR and he cannot approve it. Since VOY-212, Claude opens PRs through the **Open PR** workflow instead. The workflow runs as the `voyagami-bot` GitHub App, so the bot is the PR author and Mihir can approve.

## How it works

1. Claude pushes the ticket branch (for example `voy-212`).
2. Claude starts the `Open PR` workflow (`.github/workflows/open-pr.yml`) on `main` with `branch`, `title` and `body`.
3. The workflow mints a one-hour installation token from the app's private key and opens the PR (or updates the open PR for that branch). It assigns Mihir and requests his review.
4. CI runs as usual. A PR opened with an app token triggers `pull_request` workflows; one opened with the default `GITHUB_TOKEN` would not, which is why this uses an app.
5. Mihir reviews and approves. Branch protection blocks the merge until he does.

Where the key lives: the app's private key is stored only as a GitHub Actions secret. It is never put in the Claude cloud environment, and Claude never sees it. Claude only holds its normal GitHub connection.

Only `msriv` can start the workflow, and only from `main`.

## One-time setup (Mihir)

### 1. Create the app

GitHub, Settings, Developer settings, GitHub Apps, **New GitHub App** (https://github.com/settings/apps/new):

- Name: `voyagami-bot` (any free name works).
- Homepage URL: `https://github.com/msriv/voyagami`.
- Webhook: untick **Active**.
- Repository permissions:
  - Contents: **Read-only**
  - Pull requests: **Read and write**
  - Issues: **Read and write** (assignees)
  - Metadata: Read-only (set automatically)
- Where can this app be installed: **Only on this account**.

Create it. On the app page, note the **App ID**. Under **Private keys**, choose **Generate a private key**; a `.pem` file downloads.

### 2. Install it

On the app page, **Install App**, pick `msriv`, **Only select repositories**, choose `voyagami`.

### 3. Store the credentials in the repo

Repo `msriv/voyagami`, Settings, Secrets and variables, Actions:

- **Variables** tab: `BOT_APP_ID` = the App ID.
- **Secrets** tab: `BOT_APP_PRIVATE_KEY` = the whole contents of the `.pem` file.

Then delete the downloaded `.pem` from your machine. If it leaks, revoke it on the app page and generate a new one.

Also under Settings, Actions, General, leave **Workflow permissions** as they are; the workflow does not use `GITHUB_TOKEN` to write.

### 4. Protect `main`

Settings, Branches (or Rules, Rulesets), add a rule for `main`:

- **Require a pull request before merging**, with **Required approvals: 1**.
- **Dismiss stale pull request approvals when new commits are pushed**: on. A push after approval needs a fresh approval.
- **Require approval of the most recent reviewable push**: **off**. Claude's commits are pushed as `msriv`, so this setting would stop Mihir approving them.
- **Require status checks to pass**: optional. CI jobs that a path filter skips count as passed.
- **Do not allow bypassing the above settings** (classic rule) or no bypass list (ruleset). Without this, an admin account, which is what Claude acts as, could merge without approval.

### 5. Test it

Actions, Open PR, **Run workflow** on `main` with an existing pushed branch. The PR should show `voyagami-bot[bot]` as the author, and Mihir should see the **Approve** option.

## After setup

Once Mihir confirms the steps above are done, the merge rule in `CLAUDE.md` changes in a separate PR: Claude opens PRs through this workflow, and merges a PR itself only after Mihir has approved it and CI is green.

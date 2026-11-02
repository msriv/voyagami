#!/usr/bin/env bash
# Sets up or updates one environment end to end. Safe to re-run.
#
#   infra/scripts/apply.sh dev        # or staging, prod
#
# Run in Cloud Shell (or anywhere with gcloud and terraform signed in as a project
# Owner). Steps: state bucket, APIs and empty secrets, random values for empty
# generated secrets, then the full apply. Stops with instructions when a secret
# needs a value only a person can give.
set -euo pipefail

ENV_NAME="${1:?usage: apply.sh dev|staging|prod}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
dir="$root/infra/envs/$ENV_NAME"
[ -d "$dir" ] || { echo "no such environment: $dir" >&2; exit 1; }
cd "$dir"

tfvar() { sed -n "/variable \"$1\"/,/^}/s/.*default *= *\"\(.*\)\"/\1/p" variables.tf; }
PROJECT_ID="$(tfvar project_id)"
REGION="$(tfvar region)"
PREFIX="$(tfvar prefix || true)"

echo "== ${ENV_NAME}: project ${PROJECT_ID}, region ${REGION}"
gcloud config set project "$PROJECT_ID" >/dev/null

# 1. State bucket (the backend must exist before init).
"$root/infra/scripts/bootstrap-state.sh" "$PROJECT_ID" "$REGION"

# The budget is created on the project's billing account.
billing="$(gcloud billing projects describe "$PROJECT_ID" --format 'value(billingAccountName)' 2>/dev/null | sed 's|billingAccounts/||' || true)"
export TF_VAR_billing_account="$billing"
[ -n "$billing" ] || echo "warning: no billing account found, skipping the budget alert"

# Alerts go to the signed-in account unless ALERT_EMAIL says otherwise.
export TF_VAR_alert_email="${ALERT_EMAIL:-$(gcloud config get-value account 2>/dev/null)}"

terraform init -input=false

# 2. APIs and empty secret containers first, so values can be added.
targets=(-target=module.stack.module.secrets)
grep -q 'module "base"' main.tf && targets+=(-target=module.base.module.apis)
terraform apply -input=false -auto-approve "${targets[@]}"

# 3. Fill empty secrets. Values go straight to Secret Manager, never into Terraform.
has_value() {
  gcloud secrets versions list "$1" --project "$PROJECT_ID" --filter 'state=enabled' --limit 1 --format 'value(name)' | grep -q .
}
for s in auth-secret jobs-secret revenuecat-webhook-secret; do
  name="${PREFIX}${s}"
  if has_value "$name"; then
    echo "${name}: has a value"
  else
    openssl rand -hex 32 | tr -d '\n' | gcloud secrets versions add "$name" --project "$PROJECT_ID" --data-file=- >/dev/null
    echo "${name}: added a random value"
  fi
done
missing=0
maps="${PREFIX}google-maps-server-key"
if ! has_value "$maps"; then
  missing=1
  cat <<MSG

${maps} is empty. Create a server key restricted to "Places API (New)":
  Console, APIs & Services, Credentials, Create credentials, API key (project ${PROJECT_ID})
Then:
  printf '%s' "\$KEY" | gcloud secrets versions add ${maps} --project ${PROJECT_ID} --data-file=-
and run this script again.
MSG
fi
[ "$missing" = 0 ] || exit 1

# 4. Everything else.
terraform apply -input=false

echo
echo "== Done. RevenueCat webhook Authorization value (paste in RevenueCat if not done yet):"
echo "   Bearer \$(gcloud secrets versions access latest --secret ${PREFIX}revenuecat-webhook-secret --project ${PROJECT_ID})"

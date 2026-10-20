#!/usr/bin/env bash
# Build the API image, push it to Artifact Registry and roll Cloud Run to it.
# Used by `make deploy-dev` and by the "Deploy API (dev)" workflow (VOY-94).
#
# Needs: docker, gcloud signed in with access to the dev project. Terraform owns
# the service's shape; this only changes the image.
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-voyagami-dev-512208}"
REGION="${REGION:-asia-south1}"
REPOSITORY="${REPOSITORY:-voyagami}"
SERVICE="${SERVICE:-voyagami-api}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tag="${IMAGE_TAG:-$(git -C "$root" rev-parse --short=12 HEAD)}"
registry="${REGION}-docker.pkg.dev"
image="${registry}/${PROJECT_ID}/${REPOSITORY}/api:${tag}"

gcloud auth configure-docker "$registry" --quiet >/dev/null

echo "Building ${image}"
docker build --platform linux/amd64 -t "$image" "$root/services/api"
docker push "$image"

echo "Deploying to ${SERVICE}"
gcloud run deploy "$SERVICE" \
  --project "$PROJECT_ID" --region "$REGION" \
  --image "$image" --quiet

url="$(gcloud run services describe "$SERVICE" --project "$PROJECT_ID" --region "$REGION" --format 'value(status.url)')"
echo "Checking ${url}/health"
for attempt in 1 2 3 4 5 6; do
  if curl -fsS --max-time 10 "${url}/health"; then
    echo
    echo "Deployed ${tag} to ${url}"
    exit 0
  fi
  sleep 5
done
echo "health check did not return 200 after deploy" >&2
exit 1

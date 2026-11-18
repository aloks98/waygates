#!/usr/bin/env bash
# Called by the verified Nexus helper for the first attempt only.
# Recovery promotes the original candidate without rebuilding it.
set -euo pipefail
: "${NEXUS_PLAN:?}"
: "${NEXUS_VERSION:?}"
: "${NEXUS_SOURCE_SHA:?}"

cd "$(dirname "${BASH_SOURCE[0]}")/.."

repository=$(jq -er '.config.artifacts[] | select(.id == "image" and .kind == "oci") | .repository' "$NEXUS_PLAN")
release=$(jq -er '.releaseId | strings' "$NEXUS_PLAN")
platforms=$(jq -er '.config.artifacts[] | select(.id == "image" and .kind == "oci") | .platforms | join(",")' "$NEXUS_PLAN")
[[ "$repository" == registry.nexus.e412.in/aloks98/waygates ]] || { echo 'Unexpected image destination.' >&2; exit 1; }
[[ "$platforms" == linux/amd64,linux/arm64 ]] || { echo 'Both amd64 and arm64 are required.' >&2; exit 1; }
[[ "$release" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] || { echo 'Invalid release ID.' >&2; exit 1; }
[[ "$NEXUS_VERSION" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || { echo 'Expected a stable version without a v prefix.' >&2; exit 1; }
[[ "$NEXUS_SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid source revision.' >&2; exit 1; }

# Corepack selects packageManager from cwd before pnpm processes --dir.
(
  cd ui
  pnpm install --frozen-lockfile
  pnpm check
  pnpm typecheck
  pnpm test --run
)

# Register emulation before BuildKit discovers supported platforms.
docker run --privileged --rm tonistiigi/binfmt --install arm64

# Preserve the runner's DinD TLS connection when creating the Buildx builder.
docker_context=default
if [[ -n "${DOCKER_CERT_PATH:-}" ]]; then
  docker context inspect waygates-release >/dev/null 2>&1 || docker context create waygates-release \
    --docker "host=${DOCKER_HOST:?},ca=$DOCKER_CERT_PATH/ca.pem,cert=$DOCKER_CERT_PATH/cert.pem,key=$DOCKER_CERT_PATH/key.pem"
  docker_context=waygates-release
fi
buildkit_config=$(mktemp)
trap 'rm -f "$buildkit_config"' EXIT
cat > "$buildkit_config" <<'BUILDKIT'
[registry."docker.io"]
  mirrors = ["docker-cache.e412.in"]
BUILDKIT
docker buildx create --name waygates-release --driver docker-container --use \
  --buildkitd-config "$buildkit_config" "$docker_context"
docker buildx inspect --bootstrap

cache_dir=${WAYGATES_BUILDX_CACHE:-/tmp/.buildx-cache}
cache_from=()
if [[ -f "$cache_dir/index.json" ]]; then
  cache_from=(--cache-from "type=local,src=$cache_dir")
fi
build_args=(
  --file Dockerfile
  --build-arg "VERSION=$NEXUS_VERSION"
  --label "org.opencontainers.image.source=https://git.nexus.e412.in/aloks98/waygates"
  --label "org.opencontainers.image.revision=$NEXUS_SOURCE_SHA"
  --label "org.opencontainers.image.version=$NEXUS_VERSION"
  --provenance=false
)

# Integration tests exercise the exact release version in the combined image.
# This build also checks/compiles the React UI and Go server inside the Dockerfile.
docker buildx build "${build_args[@]}" "${cache_from[@]}" \
  --platform linux/amd64 --tag waygates-test:latest --load .
go test -count=1 ./backend/...

# Only stage the unique candidate. The helper alone promotes the final vX.Y.Z tag.
docker buildx build "${build_args[@]}" "${cache_from[@]}" \
  --platform "$platforms" --tag "$repository:nexus-$release-image" \
  --cache-to "type=local,dest=$cache_dir-new,mode=max" --push .
if [[ -d "$cache_dir-new" ]]; then
  rm -rf "$cache_dir"
  mv "$cache_dir-new" "$cache_dir"
fi

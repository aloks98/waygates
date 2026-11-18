# Releasing Waygates

[Nexus Releaser](https://releaser.nexus.e412.in) owns version selection, release
notes, tags, and publication. It does not deploy Waygates or change a running
installation. This follows the [Nexus onboarding guide](https://outline.nexus.e412.in/doc/nexus-release-projects-with-releaser-wESN5hAjYg).

## Release contract

- `nexus-release.json` uses schema 2 and requires one combined Waygates image at
  `registry.nexus.e412.in/aloks98/waygates`, for both `linux/amd64` and `linux/arm64`.
  There are no standalone binary artifacts: the image contains the server, UI,
  Caddy, and its plugins.
- `.forgejo/workflows/nexus-release.yaml` is manually dispatched by Releaser. It
  checks out the reviewed source SHA and verifies the pinned helper checksum.
- `scripts/nexus-release-build.sh` runs frontend lint, formatting, typechecks,
  and unit tests; builds a local amd64 image; runs the backend suite including
  container integration tests; then builds and stages the multi-architecture image.
  The release job uses the Docker-capable `large` runner with Go and Node setup,
  and preserves DinD TLS settings and dependency/build caches.
  Node is pinned in `.node-version`, Corepack in the workflow/Dockerfile, and
  pnpm in `ui/package.json`. UI checks run from `ui/`: Corepack resolves its
  version before pnpm processes `--dir`, so invoking it from the repository
  root can select a different version. Container installs use the frozen
  lockfile without a fallback that changes dependency resolution.
- Version metadata comes from `NEXUS_VERSION`, without a `v` prefix. The image
  candidate is `nexus-<release-id>-image`. Only the shared helper promotes the
  verified candidate to the final `vX.Y.Z` tag. There are no `latest`, minor, or
  prerelease tags.

The old release-event GHCR publisher has been removed. Creating a version tag or
publishing release notes must not start another image publisher or deployment.

## One-time setup

1. Merge the release files into `master` and ensure Forgejo Actions is enabled.
2. Set repository Actions secrets `NEXUS_PUBLISH_USERNAME=aloks98` and
   `NEXUS_PUBLISH_TOKEN` to a Forgejo token with `write:package` access. Do not use
   the automatic job token or the old GHCR token for package publication.
3. Sign in to Releaser through NexusID, choose **Add project**, and enter
   `aloks98/waygates`. Track `master`. For an existing project, **Sync from Forgejo**.
4. Prepare a draft. If an existing draft predates this setup, choose
   **Update draft to latest commit**. Review both artifact platforms, the Nexus
   image destination, workflow, and build script, then approve the configuration.

The Releaser server already has repository access; it does not require a new
repository token for Waygates. Approval and release execution remain explicit
owner actions in Releaser.

## Publish a version

Merge the changes and let CI pass. Sync the project, prepare or refresh the draft,
review its source, stable version, and notes, then choose **Release**. Keep the
helper revision in the JSON matched to the workflow's source/tag/checksum pins;
updating either reviewed file requires renewed approval.

Wait for Releaser to report **Released**, not just a successful workflow. It
independently verifies the published image and release metadata. Pull the exact
reference from the release, such as
`registry.nexus.e412.in/aloks98/waygates:v1.2.3`, and update your deployment
separately. See [the deployment guide](DEPLOYMENT.md#upgrading).

If a release fails, keep its candidate image. Use **Retry remaining** or
**Retry finalization** in Releaser when offered; these reuse the original bytes.
If the build failed before candidates existed, fix the source and prepare a new,
unused version. Do not manually dispatch this workflow, overwrite a reserved
tag, or republish a candidate under the same release ID.

## Validate release-file changes

Run `bash -n scripts/nexus-release-build.sh` and
`node --test scripts/nexus-release.test.mjs`. Most contract tests replace build
tools with recorders. With Docker available, run
`WAYGATES_TEST_DOCKER=1 node --test scripts/nexus-release.test.mjs` to also verify
the real context/builder setup using isolated client metadata. This checks the
runner's Docker endpoint, including DinD TLS, without building or publishing
images. CI enables this check on pull requests. A successful contract check does
not replace the first owner-approved end-to-end release.

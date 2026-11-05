# Working on the container image

- Follow the root `AGENTS.md`; this directory owns the agent image only.
- Keep the multi-stage build: compile with Go 1.26.7 or newer, then copy only
  the binary into the runtime image. Preserve `CGO_ENABLED=0`, `GOTOOLCHAIN=local`
  and `GOWORK=off` so builds use the image toolchain and published modules.
- Build from the repository root using the committed `go.mod` and `go.sum`.
  Keep module downloads cacheable; never copy sibling checkouts or add local
  replacements to published module requirements.
- Run as UID/GID `1000:1000`. Keep `/state` owned by that user and ensure mounted
  state is writable by UID 1000; Compose owns local mounts and filesystem settings.
- Retain CA certificates for HTTPS requests to the public Hacker News API.
  This workflow requires no HN credentials.
- Deployments supply `BUS_URL` and `STORAGE_URL` through environment variables
  or flags. Do not add connection defaults or credentials to the Dockerfile.
- Keep the executable as the entrypoint so flags, help and shutdown signals
  reach the agent directly.
- Preserve builder-platform compilation and release support for both
  `linux/amd64` and `linux/arm64`; demo Commander host images use `linux/amd64`.
- After image changes, build from the root with
  `docker build -f docker/Dockerfile -t commander-example-agent:local .`
  and smoke-check `docker run --rm commander-example-agent:local --help`.
  Use the root guidance for dedicated Compose integration validation.

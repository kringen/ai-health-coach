# Agent Guidance

## Repository Overview

- This is a Go 1.23 module using the Go 1.24.4 toolchain declared in `go.mod`.
- The server entry point is `cmd/server/main.go`.
- Backend code is organized under `pkg/`: `api` for HTTP handlers and routes, `auth` for authentication and middleware, `database` for PostgreSQL access and migrations, `models` for JSON/SQL data types, and `ollama` for the configurable AI integration.
- The UI is server-rendered HTML in `web/templates/` with vanilla JavaScript and CSS in `web/assets/`.
- PostgreSQL migrations live in `migrations/`; Kubernetes manifests live under `k8s/`; local services are described by `docker-compose.yaml` and run with Podman Compose.

## Working Conventions

- Keep changes scoped to the package or asset that owns the behavior. Preserve the existing constructor-based dependency injection and Gorilla Mux route registration patterns.
- Use standard Go formatting and naming. Run `gofmt` or `go fmt ./...` for Go changes; do not hand-format around generated output.
- HTTP handlers should authenticate through the existing auth helpers, validate request input, return an appropriate status code, and encode JSON with the existing response style.
- Database changes require a paired up/down migration when schema changes are needed. Keep user ownership checks in database/API paths so one user cannot access another user's entries.
- Preserve the custom `models.CustomDate` and `models.CustomTime` JSON and SQL behavior when changing date/time fields.
- Keep frontend changes framework-free unless the architecture is intentionally changed. Reuse existing asset loading, template, responsive layout, and API conventions.
- Do not commit credentials or local secrets. Use `.env` for local values based on `.env.example`; keep `.env` out of version control.
- AI is selected with `AI_PROVIDER=ollama` or `AI_PROVIDER=anthropic`. Keep provider-specific credentials in environment variables; never log or commit `ANTHROPIC_API_KEY`.

## Validation

Run the narrowest useful check first, then broaden it as needed:

```bash
go test ./path/to/changed/package
go test -race ./...
go vet ./...
go fmt ./...
```

The repository also provides:

```bash
make build
make test-unit
make test
make lint
make test-integration
```

`make test` delegates to `scripts/test.sh`, which runs formatting, dependency, lint, unit, security, vulnerability, Kubernetes, containerfile, benchmark, and optional integration checks. It can install missing tools, requires external services for some phases, and creates coverage/report artifacts. Prefer focused commands during iteration.

For local service work, install Podman and a Compose provider such as `podman-compose`, then run `make up`; this starts the services on port 8080 with PostgreSQL and the Nginx reverse proxy on port 8088. Rootless Podman cannot bind host ports below 1024. Use `make COMPOSE_CMD="podman compose" up` when the provider is exposed through `podman compose`. `make down` stops the services. The AI client uses the configured Ollama endpoint and is not required for ordinary model/API unit tests.

## Change Safety

- Inspect nearby tests before changing behavior and add focused regression coverage for new API, model, database, or authentication behavior.
- Do not broaden an endpoint or weaken authorization merely to make a test pass.
- Update migrations, templates, JavaScript, CSS, and documentation only when the behavior change requires it.
- Avoid destructive cleanup of existing working-tree changes or generated artifacts unless explicitly requested.
- The README and Makefile are not perfectly synchronized; verify commands against the Makefile and scripts before documenting or invoking them.
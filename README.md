# Card-to-SHEBA Service: Docker Setup

This guide runs the Go API and Nuxt frontend with the portable Docker configuration. It uses public Go, Debian, and Node images and does not require the local Go archive at `/home/star/go1.27.1.linux-amd64.tar.gz`.

## Requirements

Install:

- Docker Engine
- Docker Compose v2
- Network access for the first image build

No local Go, Node.js, npm, or SQLite installation is required.

## Configure credentials

From the repository root:

```bash
cp .env.example .env
```

Edit `.env` and set one of these values:

```dotenv
ZARINHUB_TOKEN=your-token
```

or:

```dotenv
ZARINHUB_GUID=your-guid
```

Never commit `.env` or real credentials to GitHub.

## Build and start

Run from the repository root:

```bash
docker compose -f compose.github.yaml up --build
```

The first build downloads the required base images and npm dependencies. Later starts can use the cached images:

```bash
docker compose -f compose.github.yaml up -d --no-build
```

## Open the application

Frontend:

```text
http://localhost:3000
```

Backend health check:

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

## Test the API

```bash
curl -sS http://localhost:8080/api/v1/card-to-sheba \
  -H 'Content-Type: application/json' \
  -d '{"card":"6037997599939999"}'
```

The frontend calls the Go API only. SQLite audit data is persisted in the repository's `data/` directory.

## Useful commands

Show service status:

```bash
docker compose -f compose.github.yaml ps
```

Follow logs:

```bash
docker compose -f compose.github.yaml logs -f
```

Stop the services:

```bash
docker compose -f compose.github.yaml down
```

Rebuild after source or dependency changes:

```bash
docker compose -f compose.github.yaml up --build
```

## Configuration

The portable stack uses:

| Service | Container port | Host port |
|---|---:|---:|
| Go API | 8080 | 8080 |
| Nuxt frontend | 3000 | 3000 |

The API runs migrations automatically when it starts. No separate database container is needed because SQLite is embedded in the Go application.

## Troubleshooting

If ports are already in use:

```bash
ss -ltnp '( sport = :3000 or sport = :8080 )'
```

Stop an existing Compose stack before restarting:

```bash
docker compose -f compose.github.yaml down
```

If the frontend shows a connection error, check that the API is healthy:

```bash
curl -i http://localhost:8080/healthz
```

If credentials are missing, check `.env` and restart the stack:

```bash
docker compose -f compose.github.yaml down
docker compose -f compose.github.yaml up --build
```

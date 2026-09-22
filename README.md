# Card-to-SHEBA service

Full-stack Card-to-SHEBA conversion: a Go HTTP API talks to ZarinHub, writes an SQLite audit log, and a Nuxt UI shows the result.

## Architecture

- `internal/httpapi` — HTTP transport (routing, JSON, CORS, request IDs)
- `internal/inquiry` — application service (validation + orchestration)
- `internal/zarinhub` — outbound ZarinHub client (`POST /api/v5/Kyc/CardToIban`)
- `internal/repository` — SQLite persistence
- `internal/validation` — card-number checks (digits, length, Luhn)

Handlers do not call ZarinHub. Dependencies are constructor-injected. `context.Context` is passed through, and the outbound client uses an explicit timeout.

## Setup

Requires Go 1.22+ for the API. The UI requires Node.js 20.19.0+ or 22.12.0+ to satisfy the Nuxt 3 dependency set.

```bash
cp .env.example .env
# edit .env and set ZARINHUB_TOKEN or ZARINHUB_GUID
```

`.env` is loaded automatically by `cmd/api` and must stay out of version control.

### Configuration

| Variable | Purpose |
|---|---|
| `HTTP_ADDR` | Listen address (default `:8080`) |
| `SQLITE_PATH` | SQLite file path |
| `MIGRATIONS_PATH` | Directory containing SQL migrations |
| `CORS_ORIGIN` | Allowed frontend origin |
| `ZARINHUB_BASE_URL` | `https://zarin-hub.com` |
| `ZARINHUB_TIMEOUT` | Outbound HTTP timeout (default `8s`) |
| `ZARINHUB_TOKEN` | Bearer token (`Authorization: Bearer …`) |
| `ZARINHUB_GUID` | Used as the Bearer token when `ZARINHUB_TOKEN` is empty |
| `ZARINHUB_API_PASSWORD` | Stored locally only; the documented CardToIban call uses Bearer auth |

### Migrations

Migrations run automatically when the API starts (`migrations/001_init.sql`). To apply them on their own:

```bash
mkdir -p data
sqlite3 data/app.db < migrations/001_init.sql
```

### Run the API

```bash
go run ./cmd/api
```

### Run the frontend

```bash
cd web
npm install
NUXT_PUBLIC_API_BASE=http://localhost:8080 npm run dev
```

Open `http://localhost:3000`.

## Internal API

Frontend should call only this backend. The JSON body matches the ZarinHub Card-to-IBAN contract.

### Convert card to SHEBA

`POST /api/v1/card-to-sheba`

```bash
curl -sS http://localhost:8080/api/v1/card-to-sheba \
  -H 'Content-Type: application/json' \
  -d '{"card":"6037997599939999"}'
```

Success (`200`) returns ZarinHub fields unchanged:

```json
{
  "bankName": "ملت",
  "card": "6037997599939999",
  "deposit": "0010608837001",
  "depositDescription": "فعال",
  "depositOwners": "علی محمدی",
  "depositStatus": "02",
  "englishBankName": "SAMAN",
  "iban": "IR120570028080010608837001"
}
```

Error (example):

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "card number must be 16 digits",
    "request_id": "…"
  }
}
```

| Application code | Typical HTTP status |
|---|---|
| `VALIDATION_ERROR` | 400 |
| `BAD_REQUEST` | 400 |
| `UNAUTHORIZED` | 401 |
| `ACCESS_DENIED` | 403 |
| `NOT_FOUND` | 404 |
| `LOGIC_ERROR` | 422 |
| `UNAVAILABLE` | 503 |
| `TIMEOUT` | 504 |
| `PROVIDER_ERROR` / `SERVER_ERROR` | 502 |
| `INTERNAL_ERROR` | 500 |

ZarinHub public codes (`card-to-iban-v5-*`) are mapped to the table above. Responses and logs never include credentials, stack traces, or a full unmasked card.

`GET /healthz` returns `{"status":"ok"}`.

## Tests

```bash
go test ./...
```

ZarinHub is faked in unit and HTTP tests, so they do not use live credentials.

## Assumptions

- Iranian cards are 16 digits and must pass Luhn before any outbound call.
- When `ZARINHUB_TOKEN` is empty, `ZARINHUB_GUID` is sent as the Bearer token, matching the provided sample (`Authorization: Bearer YOUR_TOKEN`).
- Successful ZarinHub JSON is stored on the audit row; failed attempts are still inserted, without treating error bodies as success payloads.
- The API does not authenticate end users; CORS is limited to `CORS_ORIGIN`.

## Known limitations

- Live conversion depends on valid ZarinHub credentials and network access to `zarin-hub.com`.
- SQLite is a single-writer store; this service is intended for a single API process.
- There is no inquiry history UI; audits are backend-only.
- Card BIN / bank-specific checks beyond Luhn are not implemented.

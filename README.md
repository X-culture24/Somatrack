# ACK St. Mary's School Kabete — School Management System

A Go microservices backend (Postgres + Redis) serving a **Flutter** mobile app (parent/teacher/student) and a **React + Vite** web frontend, covering fingerprint attendance (students and staff), fee management, payroll, and M-Pesa payments/payouts.

> This README describes the system as it actually runs today. An earlier draft of this file described a Django/React stack that was never present in this repository — ignore any references to `manage.py`, `requirements.txt`, or Celery you may find elsewhere; there is no Django here.

## Architecture

Six Go binaries share one Postgres database and talk to each other over an internal HTTP API (`X-Service-Key` header, not exposed publicly):

| Service | Port | Responsibility |
|---|---|---|
| `api-gateway` (`cmd/api`) | 8000 | Public entrypoint. Auth (JWT), dashboard, proxies to the services below, plus a large surface of still-placeholder CRUD stubs (students, guardians, classes, announcements, library, transport, Nova, …) |
| `fingerprint-svc` | 8004 | Biometric device registration/heartbeat/scan, fingerprint template storage (AES-256-GCM at rest), attendance for **both** students and staff, daily absence rollup |
| `finance-svc` | 8006 | Fee management (invoices/payments/fee-structures/discounts, read side of the M-Pesa C2B reconciler) and payroll (run generation from `staff_profiles` + `staff_deductions`, M-Pesa B2C salary disbursement) |
| `payment-svc` | 8002 | M-Pesa C2B webhook (fee payments in), fuzzy student matching, invoice reconciliation, fee reminders |
| `notification-svc` | 8003 | SMS/email dispatch (console backend by default) |
| `report-svc` | 8005 | Internal RPC client wired in; not yet exposed publicly |

A thin Django app (`mpesa_daraja_b2c/`) prototyped the M-Pesa B2C payout flow; its logic has since been **ported into `finance-svc` in Go** (`internal/finance/daraja.go`) so the whole backend stays one language. The Django folder is kept for reference but isn't part of the running stack.

```
mobile/            Flutter app (parent, teacher, student)
frontend/          React + Vite web app (admin/finance/teacher/student/parent/Nova) — not exercised in this pass, treat as unverified against the current Go API
cmd/               One main.go per Go service
internal/          Go service implementations, shared JWT/httpx/rpc/types packages
migrations/        Plain-SQL migrations, applied in filename order (000, 004–010; no golang-migrate dependency needed to run them)
mpesa_daraja_b2c/  Reference-only Django B2C prototype (superseded by internal/finance)
postman/           Postman collection covering every api-gateway route, with test assertions
deploy/nginx.conf  Reverse-proxy config for the Docker/production topology
```

## Running it locally (native, no Docker)

This is how the stack is actually run day to day in this environment — plain binaries against a local Postgres/Redis, no containers.

```bash
# 1. Postgres + Redis running locally, and a role/db matching .env:
#    role "st_mary" / db "st_marys_db" (see .env for the exact DSN)
sudo -u postgres createuser -P st_mary   # password: see DATABASE_URL in .env
sudo -u postgres createdb -O st_mary st_marys_db
psql -d st_marys_db -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"

# 2. Apply migrations (plain SQL, no external tool required)
for f in $(ls migrations/*.up.sql | sort -V); do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
done

# 3. Build and start every service (each reads ./.env via godotenv)
go build -o bin/api ./cmd/api
go build -o bin/fingerprint-svc ./cmd/fingerprint-svc
go build -o bin/payment-svc ./cmd/payment-svc
go build -o bin/notification-svc ./cmd/notification-svc
go build -o bin/report-svc ./cmd/report-svc
go build -o bin/finance-svc ./cmd/finance-svc

./bin/fingerprint-svc &
./bin/payment-svc &
./bin/notification-svc &
./bin/report-svc &
./bin/finance-svc &
./bin/api &            # start last — it depends on the others being reachable

curl http://localhost:8000/api/health/
```

`go.mod` targets Go 1.22+; `go.sum` is checked in, so `go build` works offline once dependencies are first fetched.

### Docker Compose (production-shaped alternative)

`docker-compose.yml` builds all six services plus an nginx gateway and is kept in sync with the native setup (same env vars, same ports). `docker compose up -d --build` brings up the whole stack, including `migrate` as a one-shot job. Not used for day-to-day local development in this environment, but it's what a real deployment should start from.

## Mobile app (Flutter)

```bash
cd mobile
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8000   # or -d linux, or a real device
```

`lib/core/constants/app_constants.dart` defaults `API_BASE_URL` to `http://10.0.2.2:8000` (the Android-emulator alias for the host) — override it with `--dart-define` for web/desktop/physical-device runs. Covers parent (child attendance + fee views, with an arrival-time-vs-date chart), teacher (own attendance, classes, timetable), and a student stub.

## API testing

`postman/St_Marys_LMS.postman_collection.json` covers every route registered on `api-gateway` — 288 requests, each with `pm.test()` assertions (status-code allowlist, JSON-shape checks, and negative-path tests), 1134 assertions total. Import it, run **Auth → Login** first (it has a script that auto-captures `access`/`refresh`/`user_id` into collection variables — no manual token copying), and everything else authenticates automatically.

```bash
npx newman run postman/St_Marys_LMS.postman_collection.json
```

Default collection variables point at `http://localhost:8000/api` with a bootstrap admin (`admin@test.local` / `TestPass123!`) — create your own users and delete that account before using this against anything real.

## Fingerprint attendance

- Devices register with a shared secret (`POST /fingerprint/devices/register/`, admin-only) and sign every heartbeat/scan with `HMAC-SHA256(key = HashDeviceSecret(secret), message = <canonical request content>)` — the signature covers device ID, scan type, student/staff ID, and timestamp, not just the timestamp, and requests must land within a 5-minute freshness window. (An earlier version of this signature only covered the timestamp, which allowed replaying one observed heartbeat's signature to forge scans with arbitrary attacker-chosen data — fixed; see `internal/fingerprint/crypto.go`.)
- `entry`/`late`/`exit`/`class_checkin` scans mirror into `attendance_records` for students; `entry`/`exit` scans mirror into a parallel `staff_attendance` table for staff/teachers, exposed via `GET /attendance/staff/me/range/` for self-service and `GET /attendance/staff/{staff_id}/range/` for admins/teachers.
- Guardians get an SMS on every non-duplicate student scan (arrival, late arrival, sign-out, class check-in) via `notification-svc`.
- Templates are stored AES-256-GCM-encrypted (key derived via HKDF from `FINGERPRINT_ENCRYPTION_KEY`); the raw template never appears in an API response.

## Fee management & payroll

- **Fees in**: M-Pesa C2B confirmation/validation webhooks (`payment-svc`) fuzzy-match the payer to a student, post the payment, and recompute invoice balances in a transaction. `finance-svc` exposes the read side (`GET /invoices/`, `/payments/`, `/fee-structures/`, `/discounts/`) with per-role scoping — parents/students only ever see their own or their ward's records.
- **Payroll**: `POST /payroll-runs/` (finance role) computes gross/net pay per active staff member from `staff_profiles.basic_salary` minus their active `staff_deductions` rows. **There is no statutory PAYE/NHIF/NSSF tax-band calculation** — the `paye`/`nhif`/`nssf` fields on a payroll entry only reflect amounts an admin has recorded, not a computed liability. Wire a real KRA tax-band calculator into `internal/finance/payroll.go` before relying on this for actual payslips.
- **Salary payouts**: `POST /payroll-entries/{id}/disburse/` triggers an M-Pesa B2C payment via `internal/finance/daraja.go`. It will fail with a clear error until real Safaricom Daraja credentials (`DARAJA_CONSUMER_KEY`/`_SECRET`, initiator credentials, and a production certificate) are set — see `.env.example` for the full `DARAJA_*` list.

## Configuration

See `.env.example` for the full list. The ones worth knowing about:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres DSN, shared by every service |
| `JWT_SECRET` / `DJANGO_SECRET_KEY` | Signs access/refresh JWTs. **Startup refuses to run** if `DJANGO_DEBUG=false` and this is still the checked-in default — see Security below |
| `FINGERPRINT_ENCRYPTION_KEY`/`_SALT` | Key material for template encryption at rest; same fail-closed check applies |
| `*_SERVICE_URL` / `*_SERVICE_KEY` | Inter-service base URLs and shared `X-Service-Key` secrets |
| `MPESA_SHARED_SECRET` / `MPESA_ALLOWED_IPS` | Guards the public C2B webhook |
| `DARAJA_*` | B2C payout credentials (payroll disbursement) |

## Security notes

A pass over the whole system found and fixed:

- **Fingerprint replay/forgery** — device HMAC now covers the full request, not just a bare timestamp, plus a 5-minute freshness window (see above).
- **No rate limiting on login** — `/auth/login/` and `/auth/refresh/` are now throttled per client IP (burst 5, then ~1/12s); confirmed the 6th rapid attempt gets `429`.
- **Insecure defaults usable in production** — `config.Load()` now hard-fails at startup if `DJANGO_DEBUG=false` while the JWT secret or fingerprint encryption key is still the value checked into `docker-compose.yml`/`.env.example`.
- **M-Pesa webhook IP allowlist bypass** — `MPESA_ALLOWED_IPS` used to trust the first (attacker-controlled) entry of `X-Forwarded-For`; it now prefers nginx's `X-Real-IP` and falls back to the *last* `X-Forwarded-For` hop.
- **Non-constant-time secret comparisons** — the M-Pesa shared-secret check and the inter-service `X-Service-Key` check both used timing-unsafe comparisons; both now use `crypto/subtle`.
- **Broken auth query** — `internal/api/auth.go` selected `phone_number`/`password_hash` columns that don't exist (actual columns are `phone`/`password`); login was completely broken before this was found and fixed.
- **Context-key mismatch** — the auth middleware stored `user_id`/`role` under a private typed context key while every proxy handler read them with plain string keys, so every downstream authorization check silently failed open to "not authenticated"/"forbidden". Fixed by also setting the plain-string keys.
- **Role casing mismatch** — the `users.role` column is lowercase (`system_admin`) but `types.Role` constants are uppercase (`SYSTEM_ADMIN`), so every role-based permission check silently failed. Fixed by uppercasing at the read boundary.

**Known gaps, not fixed this pass:**
- `POST /api/auth/nova/handoff/` 500s (`nova_login_tickets.role` NOT NULL violation) — a real bug in the Nova SSO handoff, untouched.
- A handful of unused, confusingly-broken RBAC helper functions remain in `internal/middleware/auth.go` (`IsAuthenticated`, `RequireRoles`, etc.) — dead code, not wired into any route, but should be deleted before someone assumes they work.
- The DB's role vocabulary (`bursar`, `nurse`, `storekeeper`, `transport_officer`, `deputy_head`) and the Go backend's `types.Role` vocabulary (`SCHOOL_DIRECTOR`, `ADMISSIONS_OFFICER`, `RECEPTIONIST`, `TRANSPORT_COORDINATOR`, `DEPUTY_HEAD_TEACHER`) don't fully overlap — needs a deliberate reconciliation, not a guessed mapping.
- Most of `api-gateway`'s CRUD surface (students, guardians, classes, announcements, transport, library, inventory, Nova, …) is still a placeholder stub returning `{"status": "placeholder_stub"}` — real logic was never ported from the original Django views for these resources. Fingerprint attendance, fee management, and payroll are the exceptions: those are real.

## Banking / M-Pesa reference

- Account name: ACK ST. MARY'S SCHOOL KABETE
- Equity Bank Kangemi `1370263402101`
- M-Pesa Paybill `247247`, account format `137101#CHILDSNAME`

```bash
curl -X POST http://localhost:8000/api/webhooks/mpesa/confirmation \
  -H "Content-Type: application/json" \
  -H "X-Mpesa-Secret: dev-mpesa-secret" \
  -d '{"TransID":"QK7TEST001","TransAmount":"5000","BillRefNumber":"137101#FAITH OTIENO","TransTime":"20260115104500"}'
```

## Production checklist

- Set real `JWT_SECRET`/`DJANGO_SECRET_KEY`, `FINGERPRINT_ENCRYPTION_KEY`, and every `*_SERVICE_KEY` — the app will refuse to boot with the defaults once `DJANGO_DEBUG=false`, but double-check they're actually rotated, not just non-default-looking.
- Fill in real `DARAJA_*` credentials before anyone tries to disburse payroll.
- Put `MPESA_ALLOWED_IPS` behind Safaricom's published IP ranges and keep `MPESA_SHARED_SECRET` set.
- Serve everything behind nginx per `deploy/nginx.conf`; don't expose the individual service ports (8002–8006) publicly — only `api-gateway` (8000) and the fingerprint device endpoints nginx proxies directly should be reachable.
- Daily `pg_dump` of Postgres.

# Daraja B2C Payment Service (Django)

A small Django + Django REST Framework service that disburses money to
M-Pesa phone numbers using Safaricom's **Daraja B2C API** (Business-to-Customer
payments — salary payments, business payments, or promotion payouts).

## What's included

- `b2c/services.py` — `DarajaB2CClient`: gets/caches an OAuth token and
  calls the `/mpesa/b2c/v1/paymentrequest` endpoint.
- `b2c/utils.py` — phone number normalisation and RSA encryption of the
  initiator password into Daraja's required `SecurityCredential` value.
- `b2c/models.py` — `B2CTransaction`, one row per payout attempt, tracking
  status from `PENDING` → `ACCEPTED` → `SUCCESS`/`FAILED`/`TIMEOUT`.
- `b2c/views.py` + `b2c/urls.py` — REST endpoints:
  - `POST /api/b2c/initiate/` — trigger a payout
  - `GET  /api/b2c/transactions/<uuid>/` — poll a transaction's status
  - `POST /api/b2c/callback/result/` — Safaricom's async result webhook
  - `POST /api/b2c/callback/timeout/` — Safaricom's async timeout webhook
- `b2c/admin.py` — read-only admin view of transactions.

## How B2C actually works (important)

Calling `initiate/` does **not** mean the payment succeeded — it only means
Safaricom accepted the request. Safaricom will asynchronously `POST` the
real outcome to your **ResultURL** (success/failure) or **QueueTimeOutURL**
(if it timed out) a few seconds to minutes later. This service records that
in `B2CTransaction`, so poll `GET /api/b2c/transactions/<uuid>/` or watch
your own database for the final status — don't treat the `initiate/`
response as final.

Because of this, `DARAJA_B2C_RESULT_URL` / `DARAJA_B2C_TIMEOUT_URL` must be
**publicly reachable HTTPS URLs** that reach this Django app. In local
development, use a tunnel such as [ngrok](https://ngrok.com/):

```bash
ngrok http 8000
# then set in .env:
# DARAJA_B2C_RESULT_URL=https://<your-ngrok-id>.ngrok-free.app/api/b2c/callback/result/
# DARAJA_B2C_TIMEOUT_URL=https://<your-ngrok-id>.ngrok-free.app/api/b2c/callback/timeout/
```

## Setup

1. **Install dependencies** (Python 3.10+ recommended):

   ```bash
   cd mpesa_daraja_b2c
   python3 -m venv .venv && source .venv/bin/activate
   pip install -r requirements.txt
   ```

2. **Configure environment variables**:

   ```bash
   cp .env.example .env
   ```

   Fill in the values from your app at
   https://developer.safaricom.co.ke (Daraja portal):
   - `DARAJA_CONSUMER_KEY` / `DARAJA_CONSUMER_SECRET` — from your app.
   - `DARAJA_INITIATOR_SHORTCODE`, `DARAJA_INITIATOR_NAME`,
     `DARAJA_INITIATOR_PASSWORD` — from the sandbox "Test Credentials"
     page (or your production go-live pack).
   - `DARAJA_B2C_RESULT_URL` / `DARAJA_B2C_TIMEOUT_URL` — your public
     callback URLs (see above).

3. **Add Safaricom's public certificate** used to encrypt the initiator
   password (see `certs/README.md` for where to get it):

   ```bash
   cp /path/to/SandboxCertificate.cer certs/SandboxCertificate.cer
   ```

4. **Run migrations and start the server**:

   ```bash
   python manage.py migrate
   python manage.py createsuperuser   # optional, for /admin/
   python manage.py runserver
   ```

## Triggering a payment

```bash
curl -X POST http://localhost:8000/api/b2c/initiate/ \
  -H "Content-Type: application/json" \
  -d '{
        "phone_number": "0712345678",
        "amount": 100,
        "remarks": "Refund for order #123",
        "occasion": "Order-123"
      }'
```

Response (`202 Accepted`):

```json
{
  "id": "b1e2c3d4-...",
  "phone_number": "254712345678",
  "amount": "100.00",
  "status": "ACCEPTED",
  "originator_conversation_id": "...",
  "conversation_id": "AG_...",
  "response_code": "0",
  "response_description": "Accept the service request successfully.",
  ...
}
```

Then poll:

```bash
curl http://localhost:8000/api/b2c/transactions/<id>/
```

Once Safaricom's callback lands, `status` becomes `SUCCESS` (with
`transaction_id`, `transaction_amount`, `receiver_public_name` filled in)
or `FAILED` / `TIMEOUT` (with `result_desc` explaining why).

## Sandbox testing notes

- Safaricom's sandbox only accepts a handful of specific test MSISDNs for
  B2C — check the "Test Credentials" page in the Daraja portal for the
  current list; other numbers will be accepted by `initiate/` but the
  callback will report a failure.
- `CommandID` must be one of `SalaryPayment`, `BusinessPayment`, or
  `PromotionPayment` — sandbox generally only exercises `BusinessPayment`.
- If you see `401` errors from Daraja, double check the consumer
  key/secret and that your sandbox app has the "M-Pesa Sandbox" product
  enabled.

## Security notes

- Never commit `.env` or the real `.cer` certificate files (`.gitignore`
  already excludes them).
- The callback endpoints (`callback/result/`, `callback/timeout/`) are
  intentionally CSRF-exempt and unauthenticated because Safaricom calls
  them directly — in production, put them behind an IP allowlist or a
  verification step appropriate to your infra (e.g. only accept requests
  from Safaricom's published IP ranges, or a reverse-proxy secret path).
- `DARAJA_API_SHARED_SECRET`, if set, requires callers of `/initiate/` to
  send a matching `X-API-KEY` header — replace with your app's real auth
  (e.g. DRF token/session auth) before going to production.

## Going to production

1. Set `DARAJA_ENV=production` and swap in your production consumer
   key/secret, shortcode, initiator credentials, and
   `ProductionCertificate.cer`.
2. Set `DJANGO_DEBUG=False`, a real `DJANGO_SECRET_KEY`, and a real
   database (Postgres, etc.) instead of SQLite.
3. Point `CACHES` at Redis/Memcached if you run more than one app
   instance, so the cached OAuth token is shared.
4. Serve over HTTPS with a real domain for the result/timeout URLs.

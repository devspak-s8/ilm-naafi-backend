# Ilm Nafi Worker (Cloudflare)

The backend rewritten as a Cloudflare Worker — fully serverless: **Workers + D1 + KV + MailChannels**. No VPS, no Docker, nothing to keep running. Go code remains in the repo root for reference/tooling.

## Stack

- **Runtime**: Cloudflare Workers (TypeScript, Hono)
- **Database**: D1 (SQLite) — schema in `migrations/0001_init.sql`
- **Rate limiting**: Workers KV (`RATE_LIMIT` namespace)
- **Email**: Cloudflare Email Workers (native send_email binding) with MailChannels fallback
- **Passwords**: PBKDF2-HMAC-SHA256 via WebCrypto (100,000 iterations — the Workers maximum)
- **Quran content**: Quran Foundation provider (OAuth2 client credentials)

## Setup

```bash
cd worker
npm install
```

### 1. Log in to Cloudflare

```bash
npx wrangler login
```

### 2. Create the resources

```bash
# D1 database — copy the printed database_id into wrangler.toml
npx wrangler d1 create ilm-naafi-db

# KV namespace for rate limiting — copy the id into wrangler.toml
npx wrangler kv namespace create RATE_LIMIT
```

Edit `wrangler.toml`: replace `REPLACE_WITH_DATABASE_ID` and `REPLACE_WITH_KV_ID`.

### 3. Configure secrets

```bash
npx wrangler secret put JWT_ACCESS_SECRET            # min 32 chars
npx wrangler secret put QURAN_FOUNDATION_CLIENT_ID   # from Quran Foundation
npx wrangler secret put QURAN_FOUNDATION_CLIENT_SECRET
```

For local dev, copy `.dev.vars.example` to `.dev.vars` and fill values.

Non-secret settings live in `[vars]` in `wrangler.toml`:
- `APP_URL` — must be your public API URL (used in email links)
- `EMAIL_FROM` / `EMAIL_FROM_NAME`
- `CORS_ALLOWED_ORIGINS` — comma-separated frontend origins
- `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_WINDOW`
- `QURAN_FOUNDATION_ENV` — `prelive` or `production`

### 4. Apply the schema

```bash
# production
npx wrangler d1 execute ilm-naafi-db --remote --file=./migrations/0001_init.sql

# local
npx wrangler d1 execute ilm-naafi-db --local --file=./migrations/0001_init.sql
```

### 5. Deploy

```bash
npm run deploy
```

This publishes the Worker. It serves on `*.workers.dev` immediately.

### 6. Attach your domain

In the Cloudflare dashboard (or wrangler): add a route for `api.ilmnafi.com` →
`ilm-naafi-backend` Worker. If you already created a Tunnel CNAME for
`api.ilmnafi.com`, delete that DNS record first — the Worker replaces it.

## Email (Cloudflare-first)

Email is sent through Cloudflare's own email path:

1. **Email Workers** (primary) — the native `send_email` binding already wired
   in `wrangler.toml`. Requires the **Workers Paid plan**. In the dashboard,
   enable **Workers & Pages → ilm-naafi-backend → Settings → Email Workers**
   and make sure `noreply@ilmnafi.com` is on a domain in your account.
2. **MailChannels** (automatic fallback) — used when the binding is absent or
   a send is rejected. Needs these DNS records on the sending domain:
   - **SPF**: `TXT` → `v=spf1 include:relay.mailchannels.net -all`
   - **DKIM**: `CNAME` `mailchannels._domainkey` → `dkim.mailchannels.net`

## Free vs paid plan

PBKDF2 password hashing needs more CPU than the free tier allows per request
(10 ms). For development you can lower `PBKDF2_ITERATIONS` to `10000`; for
production use the **Workers Paid plan ($5/month)**. Cloudflare's WebCrypto
caps PBKDF2 at 100,000 iterations — the code clamps to that ceiling
automatically.

## API surface

Same as the Go backend, plus the dua module which was never wired in Go:

- `/auth/*` — register, login, logout, refresh, verify-email,
  resend-verification, forgot-password, reset-password, me, sessions,
  delete-account
- `/user/*` — me, profile, delete-account, sessions
- `/quran/*` — surahs, ayahs, juz, search, reciters, audio, bookmarks,
  progress, continue
- `/api/v1/quran/*` — Quran Foundation content (chapters, verses, juz,
  pages, search, resources)
- `/adhkar/*` — categories, adhkar, completions, progress, reminders
- `/dua/*` — categories, duas (newly wired)
- `/health`, `/ready`

Response envelope: `{ "success": true, "data": ... }` /
`{ "success": false, "error": { "code", "message" } }`.

## Local development

```bash
npm run db:apply:local
npm run dev           # http://localhost:8787
```

## Seeding content data

The **full Hisnul Muslim (Fortress of the Muslim) book** is available as a
seed — 132 chapters, 267 adhkar with Arabic text, English translation,
transliteration (85%), repeat counts, audio URLs, and source references
(83%). Generated from MIT-licensed datasets (asellam/HisnElMuslim,
YousefAsalya/Islamic-Pro-azkar-API, uthumany/hisn-al-muslim-api).

```bash
# production
npx wrangler d1 execute ilm-naafi-db --remote --file=./seed/hisnul_muslim_seed.sql

# local
npx wrangler d1 execute ilm-naafi-db --local --file=./seed/hisnul_muslim_seed.sql
```

To rebuild the seed SQL after changing the source datasets:

```bash
python build_seed.py   # (script kept with the datasets; see repo history)
```

## Differences from the Go version

- Passwords use PBKDF2 instead of argon2id. Legacy argon2id hashes can't be
  verified in Workers — pre-existing users must use "forgot password".
- Refresh tokens are now correctly stored (the Go version never persisted
  `refresh_token_hash`, so `/auth/refresh` was always broken); refresh
  rotates sessions as originally intended.
- The `refresh_tokens` table was dropped (unused by the Go code).
- `dua` endpoints are wired (they existed in Go but were never routed).
- `audio` and `notifications` service stubs were not ported (they were not
  routed in Go either).
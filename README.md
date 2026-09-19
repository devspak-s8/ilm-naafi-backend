# Ilm Nafi Backend

Production-ready authentication backend for the Ilm Nafi Islamic learning platform.

## Tech Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL 16
- **Authentication**: JWT (access + refresh tokens)
- **Password Hashing**: Argon2id
- **Routing**: Gorilla Mux
- **Configuration**: Viper (environment variables)
- **Migrations**: golang-migrate

## Project Structure

```
cmd/server/                    # Application entry point
internal/
  auth/
    handler/                   # HTTP handlers
    service/                   # Business logic
    repository/                # Data access
    model/                     # DTOs and types
    validator/                 # Validation rules
  user/
    handler/                   # User HTTP handlers
    service/                   # User business logic
    repository/                # User data access
    model/                     # User types
  middleware/                   # Auth, rate limiting, logging
  email/                       # Email service abstraction
  security/                    # Password hashing, tokens
  database/                    # DB connection and migrations
  config/                      # Configuration management
  health/                      # Health checks
migrations/                    # SQL migration files
```

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Docker & Docker Compose (optional)

### Using Docker (Recommended)

1. Copy `.env.example` to `.env` and configure it
2. Run:
   ```bash
   docker compose up
   ```

### Manual Setup

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Set environment variables (see `.env.example`)

3. Run migrations:
   ```bash
   # The app runs migrations automatically on startup
   ```

4. Start the server:
   ```bash
   go run ./cmd/server
   ```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `JWT_ACCESS_SECRET` | Yes | Secret for signing access tokens (min 32 chars) |
| `JWT_REFRESH_SECRET` | Yes | Secret for signing refresh tokens (min 32 chars) |
| `JWT_ISSUER` | Yes | JWT issuer claim |
| `EMAIL_PROVIDER` | Yes | Email provider: `smtp`, `sendgrid`, `mailgun` |
| `EMAIL_SMTP_HOST` | Yes* | SMTP server host |
| `EMAIL_SMTP_PORT` | Yes* | SMTP server port |
| `EMAIL_SMTP_USER` | Yes* | SMTP username |
| `EMAIL_SMTP_PASS` | Yes* | SMTP password |
| `EMAIL_FROM` | Yes | Sender email address |
| `EMAIL_FROM_NAME` | No | Sender display name |
| `APP_URL` | Yes | Application base URL |
| `SERVER_PORT` | No | Server port (default: 8080) |
| `ACCESS_TOKEN_EXPIRATION` | No | Access token TTL (default: 15m) |
| `REFRESH_TOKEN_EXPIRATION` | No | Refresh token TTL (default: 168h) |
| `RATE_LIMIT_REQUESTS` | No | Rate limit requests per window (default: 100) |
| `RATE_LIMIT_WINDOW` | No | Rate limit window (default: 1m) |
| `QURAN_PROVIDER` | No | Quran content provider (default: `quran_foundation`) |
| `QURAN_FOUNDATION_ENV` | No | `prelive` for development or `production` for approved production access |
| `QURAN_FOUNDATION_CLIENT_ID` | No | Server-side Quran Foundation client ID |
| `QURAN_FOUNDATION_CLIENT_SECRET` | No | Server-side Quran Foundation client secret |
| `QURAN_FOUNDATION_API_BASE_URL` | No | Optional API URL override, primarily for tests |
| `QURAN_FOUNDATION_OAUTH_BASE_URL` | No | Optional OAuth URL override, primarily for tests |
| `QURAN_PROVIDER_TIMEOUT` | No | Provider HTTP timeout (default: 15s) |

## API Endpoints

### Authentication

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/auth/register` | Register new user | No |
| POST | `/auth/login` | Login | No |
| POST | `/auth/logout` | Logout current session | Yes |
| POST | `/auth/refresh` | Refresh access token | No |
| POST | `/auth/verify-email` | Verify email address | No |
| POST | `/auth/resend-verification` | Resend verification email | No |
| POST | `/auth/forgot-password` | Request password reset | No |
| POST | `/auth/reset-password` | Reset password | No |
| GET | `/auth/me` | Get current user | Yes |
| GET | `/auth/sessions` | List active sessions | Yes |
| DELETE | `/auth/delete-account` | Delete account | Yes |

### User

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/user/me` | Get current user profile | Yes |
| PUT | `/user/profile` | Update profile | Yes |
| DELETE | `/user/delete-account` | Delete account | Yes |
| GET | `/user/sessions` | List sessions | Yes |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

### Quran Foundation content API

The backend exposes a provider-neutral Quran contract under `/api/v1/quran`. It keeps Quran Foundation credentials server-side and requests content through the `QuranProvider` abstraction. Configure a Backend/server application in the Quran Foundation Developer Console before using the provider. Pre-live content is limited by Quran Foundation to a small test dataset; production access is required for the complete Quran.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/quran/chapters` | List chapters; accepts `language` |
| GET | `/api/v1/quran/chapters/{chapter}` | Get normalized chapter metadata |
| GET | `/api/v1/quran/chapters/{chapter}/verses` | Get chapter verses with optional words, translations, tafsir, audio, and pagination |
| GET | `/api/v1/quran/verses/{verseKey}` | Get one verse, for example `1:1` |
| GET | `/api/v1/quran/juz/{juz}` | Get Juz metadata and verse boundaries |
| GET | `/api/v1/quran/pages/{page}` | Get verses for a Mushaf page |
| GET | `/api/v1/quran/search?q=` | Search Quran content through the documented search API |
| GET | `/api/v1/quran/resources/translations` | List translation resources |
| GET | `/api/v1/quran/resources/tafsirs` | List tafsir resources |
| GET | `/api/v1/quran/resources/recitations` | List recitation resources |

Quran text is returned from the provider without LLM generation or rewriting. Responses retain provider identifiers and provenance fields. Related Hadith references, audio endpoints, local content synchronization, and user Quran preferences are planned follow-up slices.

The provider uses OAuth2 client credentials on the backend, caches access tokens until shortly before expiry, retries a safe GET once for a 401 token refresh, and applies bounded retry behavior for 429/5xx responses. The frontend must call Ilm Nafi rather than Quran Foundation directly.

## Request/Response Examples

### Register

```bash
POST /auth/register
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "TestPass123!"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "john@example.com",
      "email_verified": false,
      "name": "John Doe",
      "status": "active"
    },
    "tokens": {
      "access_token": "jwt-token",
      "refresh_token": "random-string",
      "expires_in": 900,
      "token_type": "Bearer"
    }
  }
}
```

### Login

```bash
POST /auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "TestPass123!"
}
```

### Get Current User

```bash
GET /auth/me
Authorization: Bearer <access_token>
```

### Refresh Token

```bash
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "previous-refresh-token"
}
```

## Error Responses

```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid email or password."
  }
}
```

## Database Schema

- `users` - User accounts
- `user_profiles` - User profile data
- `sessions` - Active sessions
- `refresh_tokens` - Refresh token records
- `email_verifications` - Email verification tokens
- `password_resets` - Password reset tokens
- `audit_events` - Security audit log

## Security Features

- Argon2id password hashing
- JWT access tokens (15 min default)
- Refresh token rotation
- Secure random token generation
- Rate limiting on sensitive endpoints
- Email verification
- Password reset with expiring tokens
- Session management
- Audit logging
- Security headers

## Running Tests

```bash
go test ./...
```

## Building

```bash
go build ./...
```

## Docker

```bash
docker compose up
```

Services:
- PostgreSQL on port 5432
- Backend API on port 8080

## License

Proprietary - Ilm Nafi Platform

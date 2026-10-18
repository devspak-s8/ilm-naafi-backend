export interface Env {
	DB: D1Database;
	RATE_LIMIT?: KVNamespace;
	SEND_EMAIL?: SendEmail;
	JWT_ACCESS_SECRET: string;
	JWT_ISSUER: string;
	ACCESS_TOKEN_EXPIRATION: string;
	REFRESH_TOKEN_EXPIRATION: string;
	APP_URL: string;
	EMAIL_FROM: string;
	EMAIL_FROM_NAME: string;
	CORS_ALLOWED_ORIGINS: string;
	RATE_LIMIT_REQUESTS: string;
	RATE_LIMIT_WINDOW: string;
	PBKDF2_ITERATIONS: string;
	QURAN_PROVIDER: string;
	QURAN_FOUNDATION_ENV: string;
	QURAN_FOUNDATION_CLIENT_ID?: string;
	QURAN_FOUNDATION_CLIENT_SECRET?: string;
	QURAN_FOUNDATION_API_BASE_URL?: string;
	QURAN_FOUNDATION_OAUTH_BASE_URL?: string;
	QURAN_PROVIDER_TIMEOUT: string;
	PASSKEY_RP_ID: string;
	PASSKEY_RP_NAME: string;
	PASSKEY_ORIGIN: string;
	VAPID_PUBLIC_KEY: string;
	VAPID_PRIVATE_JWK: string;
	SUNNAH_API_KEY?: string;
}

export type AppEnv = {
	Bindings: Env;
	Variables: {
		userId: string;
		sessionId: string;
	};
};

export function intEnv(value: string | undefined, fallback: number): number {
	if (!value) return fallback;
	const n = parseInt(value, 10);
	return Number.isFinite(n) ? n : fallback;
}
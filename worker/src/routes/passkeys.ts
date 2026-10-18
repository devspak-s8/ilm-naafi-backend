import { Hono } from "hono";
import type { AppEnv, Env } from "../lib/env";
import {
	generateAuthenticationOptions,
	generateRegistrationOptions,
	verifyAuthenticationResponse,
	verifyRegistrationResponse,
} from "@simplewebauthn/server";
import { nowISO, randomToken, tokenHash, toB64Url, fromB64Url } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { created, error, ok } from "../lib/response";
import { signAccessToken } from "../lib/jwt";
import { intEnv } from "../lib/env";
import { clientIP, userAgent } from "../lib/request";

/**
 * Passkeys (WebAuthn). The RP ID must match the frontend origin's
 * registrable domain (e.g. ilm-naafi.com) and PASSKEY_ORIGIN must be the
 * frontend origin (https://ilm-naafi.com).
 */

export const passkeyRoutes = new Hono<AppEnv>();

interface PasskeyRow {
	id: string;
	user_id: string;
	public_key: string;
	counter: number;
	transports: string | null;
	device_name: string | null;
	created_at: string;
	last_used_at: string | null;
}

const CHALLENGE_TTL = 300; // seconds

async function storeChallenge(env: Env, key: string, challenge: string): Promise<void> {
	if (!env.RATE_LIMIT) return; // KV binding reused for challenge storage
	await env.RATE_LIMIT.put(`passkey:${key}`, challenge, { expirationTtl: CHALLENGE_TTL });
}

async function takeChallenge(env: Env, key: string): Promise<string | null> {
	if (!env.RATE_LIMIT) return null;
	return (await env.RATE_LIMIT.get(`passkey:${key}`)) ?? null;
}

// ---- registration -----------------------------------------------------

// POST /auth/passkey/register/options  (auth)
passkeyRoutes.post("/register/options", authRequired, async (c) => {
	const env = c.env;
	const user = (await env.DB.prepare("SELECT email FROM users WHERE id = ? AND deleted_at IS NULL")
		.bind(c.get("userId"))
		.first()) as { email: string } | null;
	if (!user) return error(c, 404, "USER_NOT_FOUND", "User not found");

	const rpID = env.PASSKEY_RP_ID || new URL(env.APP_URL).hostname;
	const options = await generateRegistrationOptions({
		rpID,
		rpName: env.PASSKEY_RP_NAME || "Ilm Nafi",
		userID: Uint8Array.from(new TextEncoder().encode(c.get("userId"))),
		userName: user.email,
		userDisplayName: user.email,
		attestationType: "none",
		authenticatorSelection: {
			residentKey: "preferred",
			userVerification: "preferred",
		},
	});

	await storeChallenge(env, `reg:${c.get("userId")}`, options.challenge);
	return ok(c, options);
});

// POST /auth/passkey/register/verify  (auth)
passkeyRoutes.post("/register/verify", authRequired, async (c) => {
	const env = c.env;
	const userId = c.get("userId");

	let body: Record<string, unknown>;
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const response = body.response as Record<string, unknown> | undefined;
	const deviceName = typeof body.device_name === "string" ? body.device_name : null;
	if (!response) return error(c, 400, "INVALID_REQUEST", "Missing attestation response");

	const expectedChallenge = await takeChallenge(env, `reg:${userId}`);
	if (!expectedChallenge) return error(c, 400, "PASSKEY_CHALLENGE_EXPIRED", "Passkey challenge expired, please try again");

	const rpID = env.PASSKEY_RP_ID || new URL(env.APP_URL).hostname;
	const expectedOrigin = env.PASSKEY_ORIGIN || "https://" + rpID;

	let verification;
	try {
		verification = await verifyRegistrationResponse({
			response: response as never,
			expectedChallenge,
			expectedOrigin,
			expectedRPID: rpID,
		});
	} catch (e) {
		console.error("passkey registration verification failed", e);
		return error(c, 400, "PASSKEY_VERIFICATION_FAILED", "Passkey verification failed");
	}

	const { verified, registrationInfo } = verification;
	if (!verified || !registrationInfo) {
		return error(c, 400, "PASSKEY_VERIFICATION_FAILED", "Passkey verification failed");
	}

	const { credential } = registrationInfo;
	const transports = credential.transports ?? [];
	await env.DB.prepare(
		`INSERT OR REPLACE INTO passkeys (id, user_id, public_key, counter, transports, device_name, created_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
		.bind(
			credential.id,
			userId,
			toB64Url(credential.publicKey),
			credential.counter,
			JSON.stringify(transports),
			deviceName || "Passkey",
			nowISO(),
			null,
		)
		.run();

	return created(c, { status: "ok" });
});

// ---- authentication ---------------------------------------------------

// POST /auth/passkey/login/options
passkeyRoutes.post("/login/options", async (c) => {
	const env = c.env;
	const rpID = env.PASSKEY_RP_ID || new URL(env.APP_URL).hostname;
	const options = await generateAuthenticationOptions({ rpID, userVerification: "preferred" });
	await storeChallenge(env, "login", options.challenge);
	return ok(c, options);
});

// POST /auth/passkey/login/verify
passkeyRoutes.post("/login/verify", async (c) => {
	const env = c.env;
	let body: Record<string, unknown>;
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const response = body.response as Record<string, unknown> | undefined;
	if (!response) return error(c, 400, "INVALID_REQUEST", "Missing assertion response");

	const expectedChallenge = await takeChallenge(env, "login");
	if (!expectedChallenge) return error(c, 400, "PASSKEY_CHALLENGE_EXPIRED", "Passkey challenge expired, please try again");

	const rawId = typeof response.rawId === "string" ? response.rawId : null;
	if (!rawId) return error(c, 400, "INVALID_REQUEST", "Missing credential id");

	const credential = (await env.DB.prepare(
		"SELECT id, user_id, public_key, counter, transports, device_name, created_at, last_used_at FROM passkeys WHERE id = ?",
	)
		.bind(rawId)
		.first()) as PasskeyRow | null;
	if (!credential) return error(c, 404, "PASSKEY_NOT_FOUND", "Passkey not found");

	const user = (await env.DB.prepare("SELECT id, status FROM users WHERE id = ? AND deleted_at IS NULL")
		.bind(credential.user_id)
		.first()) as { id: string; status: string } | null;
	if (!user) return error(c, 404, "USER_NOT_FOUND", "User not found");
	if (user.status !== "active") return error(c, 403, "ACCOUNT_DISABLED", "Your account has been disabled");

	const rpID = env.PASSKEY_RP_ID || new URL(env.APP_URL).hostname;
	const expectedOrigin = env.PASSKEY_ORIGIN || "https://" + rpID;

	let verification;
	try {
		verification = await verifyAuthenticationResponse({
			response: response as never,
			expectedChallenge,
			expectedOrigin,
			expectedRPID: rpID,
			credential: {
				id: credential.id,
				publicKey: fromB64Url(credential.public_key),
				counter: credential.counter,
				transports: credential.transports ? (JSON.parse(credential.transports) as string[]) : undefined,
			},
		});
	} catch (e) {
		console.error("passkey login verification failed", e);
		return error(c, 401, "PASSKEY_VERIFICATION_FAILED", "Passkey verification failed");
	}

	const { verified, authenticationInfo } = verification;
	if (!verified || !authenticationInfo) {
		return error(c, 401, "PASSKEY_VERIFICATION_FAILED", "Passkey verification failed");
	}

	await env.DB.prepare("UPDATE passkeys SET counter = ?, last_used_at = ? WHERE id = ?")
		.bind(authenticationInfo.newCounter, nowISO(), credential.id)
		.run();

	// Create a session (same as password login)
	const sessionId = crypto.randomUUID();
	const refreshToken = randomToken();
	const refreshHash = tokenHash(refreshToken);
	await env.DB.prepare(
		`INSERT INTO sessions (id, user_id, device_name, device_type, ip_address, user_agent, refresh_token_hash, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
		.bind(
			sessionId,
			user.id,
			credential.device_name || "Passkey",
			"passkey",
			clientIP(c),
			userAgent(c),
			refreshHash,
			nowISO(),
			new Date(Date.now() + intEnv(env.REFRESH_TOKEN_EXPIRATION, 7 * 24 * 3600) * 1000).toISOString(),
		)
		.run();

	const accessToken = await signAccessToken(env.JWT_ACCESS_SECRET, {
		sub: user.id,
		session_id: sessionId,
		iss: env.JWT_ISSUER,
		iat: Math.floor(Date.now() / 1000),
		exp: Math.floor(Date.now() / 1000) + intEnv(env.ACCESS_TOKEN_EXPIRATION, 900),
	});

	return ok(c, {
		tokens: {
			access_token: accessToken,
			refresh_token: refreshToken,
			expires_in: intEnv(env.ACCESS_TOKEN_EXPIRATION, 900),
			token_type: "Bearer",
		},
	});
});

// GET /auth/passkeys  (auth)
passkeyRoutes.get("/", authRequired, async (c) => {
	const res = await c.env.DB.prepare(
		"SELECT id, device_name, created_at, last_used_at FROM passkeys WHERE user_id = ? ORDER BY created_at DESC",
	)
		.bind(c.get("userId"))
		.all();
	return ok(c, res.results);
});

// DELETE /auth/passkeys/:id  (auth)
passkeyRoutes.delete("/:id", authRequired, async (c) => {
	const id = c.req.param("id");
	if (!id) return error(c, 400, "VALIDATION_ERROR", "Invalid passkey id");
	await c.env.DB.prepare("DELETE FROM passkeys WHERE id = ? AND user_id = ?").bind(id, c.get("userId")).run();
	return ok(c, { status: "deleted" });
});
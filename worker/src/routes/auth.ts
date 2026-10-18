import { Hono } from "hono";
import type { AppEnv, Env } from "../lib/env";
import { intEnv } from "../lib/env";
import { hashPassword, isValidEmail, nowISO, randomToken, tokenHash, validatePassword, verifyPassword } from "../lib/security";
import { signAccessToken } from "../lib/jwt";
import { sendPasswordResetEmail, sendSecurityNotification, sendVerificationEmail, sendWelcomeEmail } from "../lib/email";
import { clientIP, userAgent } from "../lib/request";
import { createNotification } from "./notifications";
import { authRequired } from "../middleware/auth";
import { rateLimit } from "../middleware/rate-limit";
import { created, error, ok } from "../lib/response";

interface UserRow {
	id: string;
	email: string;
	password_hash: string;
	email_verified: number;
	status: string;
	last_login_at: string | null;
	created_at: string;
	updated_at: string;
	deleted_at: string | null;
}

interface SessionRow {
	id: string;
	user_id: string;
	device_name: string;
	device_type: string;
	ip_address: string;
	user_agent: string | null;
	created_at: string;
	expires_at: string;
	revoked_at: string | null;
}

const USER_COLS = "id, email, password_hash, email_verified, status, last_login_at, created_at, updated_at, deleted_at";
const SESSION_COLS = "id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at, revoked_at";

function userResponse(u: UserRow, name: string) {
	return {
		id: u.id,
		email: u.email,
		email_verified: !!u.email_verified,
		name,
		status: u.status,
		last_login_at: u.last_login_at ?? undefined,
		created_at: u.created_at,
	};
}

export const authRoutes = new Hono<AppEnv>();

// ---- helpers ---------------------------------------------------------

async function getUserByEmail(env: Env, email: string): Promise<UserRow | null> {
	return (await env.DB.prepare(`SELECT ${USER_COLS} FROM users WHERE email = ? AND deleted_at IS NULL`)
		.bind(email.toLowerCase())
		.first()) as UserRow | null;
}

async function getUserByID(env: Env, id: string): Promise<UserRow | null> {
	return (await env.DB.prepare(`SELECT ${USER_COLS} FROM users WHERE id = ? AND deleted_at IS NULL`).bind(id).first()) as UserRow | null;
}

async function getProfile(env: Env, userId: string): Promise<{ name: string } | null> {
	return (await env.DB.prepare("SELECT name FROM user_profiles WHERE user_id = ?").bind(userId).first()) as { name: string } | null;
}

async function audit(env: Env, userId: string, eventType: string, ip: string, ua: string): Promise<void> {
	await env.DB.prepare(
		"INSERT INTO audit_events (id, user_id, event_type, ip_address, user_agent, created_at) VALUES (?, ?, ?, ?, ?, ?)",
	)
		.bind(crypto.randomUUID(), userId, eventType, ip, ua, nowISO())
		.run();
}

async function createSession(
	env: Env,
	userId: string,
	deviceName: string,
	deviceType: string,
	ip: string,
	ua: string,
) {
	const sessionId = crypto.randomUUID();
	const refreshToken = randomToken(32);
	const refreshHash = tokenHash(refreshToken);
	const now = nowISO();
	const expiresAt = new Date(Date.now() + intEnv(env.REFRESH_TOKEN_EXPIRATION, 7 * 24 * 3600) * 1000).toISOString();

	await env.DB.prepare(
		`INSERT INTO sessions (id, user_id, device_name, device_type, ip_address, user_agent, refresh_token_hash, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
		.bind(sessionId, userId, deviceName || "Unknown", deviceType || "web", ip, ua, refreshHash, now, expiresAt)
		.run();

	const accessToken = await signAccessToken(env.JWT_ACCESS_SECRET, {
		sub: userId,
		session_id: sessionId,
		iss: env.JWT_ISSUER,
		iat: Math.floor(Date.now() / 1000),
		exp: Math.floor(Date.now() / 1000) + intEnv(env.ACCESS_TOKEN_EXPIRATION, 900),
	});

	return {
		access_token: accessToken,
		refresh_token: refreshToken,
		expires_in: intEnv(env.ACCESS_TOKEN_EXPIRATION, 900),
		token_type: "Bearer",
	};
}

// ---- POST /auth/register ---------------------------------------------

authRoutes.post("/register", async (c) => {
	let body: { name?: string; email?: string; password?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}

	const name = (body.name ?? "").trim();
	const email = (body.email ?? "").trim();
	const password = body.password ?? "";

	if (name.length < 2 || name.length > 100) {
		return error(c, 400, "VALIDATION_ERROR", "name must be between 2 and 100 characters");
	}
	if (!isValidEmail(email)) {
		return error(c, 400, "VALIDATION_ERROR", "invalid email format");
	}
	if (password.length < 8) {
		return error(c, 400, "VALIDATION_ERROR", "password must be at least 8 characters");
	}
	// Parity with the Go handler: password_confirmation query param must match.
	const confirmation = c.req.query("password_confirmation");
	if (confirmation !== undefined && confirmation !== password) {
		return error(c, 400, "PASSWORD_MISMATCH", "Passwords do not match");
	}

	const env = c.env;
	const existing = await getUserByEmail(env, email);
	if (existing) {
		return error(c, 409, "EMAIL_ALREADY_EXISTS", "An account with this email already exists");
	}

	const passwordHash = await hashPassword(password, intEnv(env.PBKDF2_ITERATIONS, 310000));
	const userId = crypto.randomUUID();
	const now = nowISO();

	try {
		await env.DB.prepare(
			`INSERT INTO users (id, email, password_hash, email_verified, status, created_at, updated_at)
			 VALUES (?, ?, ?, 0, 'active', ?, ?)`,
		)
			.bind(userId, email.toLowerCase(), passwordHash, now, now)
			.run();
		await env.DB.prepare(
			"INSERT INTO user_profiles (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		)
			.bind(crypto.randomUUID(), userId, name, now, now)
			.run();
	} catch (e) {
		console.error("register insert failed", e);
		return error(c, 500, "INTERNAL_ERROR", "Failed to create account");
	}

	const token = randomToken(32);
	await env.DB.prepare(
		"INSERT INTO email_verifications (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
	)
		.bind(crypto.randomUUID(), userId, tokenHash(token), new Date(Date.now() + 24 * 3600 * 1000).toISOString(), now)
		.run();
	await audit(env, userId, "registration", clientIP(c), userAgent(c));
	c.executionCtx.waitUntil(sendVerificationEmail(env, email, token));
	c.executionCtx.waitUntil(sendWelcomeEmail(env, email, name));
	c.executionCtx.waitUntil(
		createNotification(
			env,
			userId,
			"welcome",
			"Welcome to Ilm Nafi",
			"Your account has been created. Verify your email to get started.",
		),
	);

	let tokens;
	try {
		tokens = await createSession(env, userId, "", "", clientIP(c), userAgent(c));
	} catch (e) {
		console.error("session create failed", e);
		return error(c, 500, "INTERNAL_ERROR", "Failed to create session");
	}

	return created(c, {
		user: userResponse({ id: userId, email: email.toLowerCase(), password_hash: passwordHash, email_verified: 0, status: "active", last_login_at: null, created_at: now, updated_at: now, deleted_at: null }, name),
		tokens,
	});
});

// ---- POST /auth/login ------------------------------------------------

authRoutes.post("/login", rateLimit("login"), async (c) => {
	let body: { email?: string; password?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const email = (body.email ?? "").trim();
	const password = body.password ?? "";
	if (!isValidEmail(email) || !password) {
		return error(c, 400, "VALIDATION_ERROR", "email and password are required");
	}

	const env = c.env;
	const ip = clientIP(c);
	const ua = userAgent(c);

	const user = await getUserByEmail(env, email);
	if (!user) {
		return error(c, 401, "INVALID_CREDENTIALS", "Invalid email or password");
	}
	if (user.status !== "active") {
		return error(c, 403, "ACCOUNT_DISABLED", "Your account has been disabled");
	}
	const valid = await verifyPassword(password, user.password_hash);
	if (!valid) {
		await audit(env, user.id, "login_failed", ip, ua);
		return error(c, 401, "INVALID_CREDENTIALS", "Invalid email or password");
	}

	const now = nowISO();
	await env.DB.prepare("UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?").bind(now, now, user.id).run();
	const profile = await getProfile(env, user.id);
	const tokens = await createSession(env, user.id, "", "", ip, ua);
	await audit(env, user.id, "login_success", ip, ua);

	return ok(c, {
		user: userResponse(user, profile?.name ?? ""),
		tokens,
	});
});

// ---- POST /auth/logout -----------------------------------------------

authRoutes.post("/logout", authRequired, async (c) => {
	await c.env.DB.prepare("UPDATE sessions SET revoked_at = ? WHERE id = ?").bind(nowISO(), c.get("sessionId")).run();
	return ok(c, undefined);
});

// ---- POST /auth/refresh ----------------------------------------------

authRoutes.post("/refresh", rateLimit("refresh"), async (c) => {
	let body: { refresh_token?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const refreshToken = body.refresh_token ?? "";
	if (!refreshToken) {
		return error(c, 400, "VALIDATION_ERROR", "refresh_token is required");
	}

	const env = c.env;
	const hash = tokenHash(refreshToken);
	const session = (await env.DB.prepare(
		`SELECT ${SESSION_COLS} FROM sessions WHERE refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?`,
	)
		.bind(hash, nowISO())
		.first()) as SessionRow | null;
	if (!session) {
		return error(c, 401, "INVALID_REFRESH_TOKEN", "Invalid refresh token");
	}

	// Rotate: revoke the old session and issue a fresh pair.
	await env.DB.prepare("UPDATE sessions SET revoked_at = ? WHERE id = ?").bind(nowISO(), session.id).run();
	const tokens = await createSession(env, session.user_id, session.device_name, session.device_type, session.ip_address, session.user_agent ?? "");
	await audit(env, session.user_id, "token_refresh", session.ip_address, session.user_agent ?? "");

	return ok(c, tokens);
});

// ---- POST /auth/verify-email -----------------------------------------

authRoutes.post("/verify-email", async (c) => {
	let body: { token?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const token = body.token ?? "";
	if (!token) {
		return error(c, 400, "VALIDATION_ERROR", "token is required");
	}

	const env = c.env;
	const hash = tokenHash(token);
	const ev = (await env.DB.prepare(
		"SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM email_verifications WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?",
	)
		.bind(hash, nowISO())
		.first()) as { user_id: string } | null;
	if (!ev) {
		return error(c, 400, "INVALID_TOKEN", "Invalid or expired verification token");
	}

	await env.DB.prepare("UPDATE users SET email_verified = 1, updated_at = ? WHERE id = ?").bind(nowISO(), ev.user_id).run();
	await env.DB.prepare("UPDATE email_verifications SET used_at = ? WHERE user_id = ? AND used_at IS NULL").bind(nowISO(), ev.user_id).run();
	await audit(env, ev.user_id, "email_verified", clientIP(c), userAgent(c));

	return ok(c, undefined);
});

// ---- POST /auth/resend-verification ----------------------------------

authRoutes.post("/resend-verification", rateLimit("resend-verification"), async (c) => {
	let body: { email?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const email = (body.email ?? "").trim();
	if (!isValidEmail(email)) {
		return error(c, 400, "VALIDATION_ERROR", "invalid email format");
	}

	const env = c.env;
	const user = await getUserByEmail(env, email);
	if (user && !user.email_verified) {
		await env.DB.prepare("UPDATE email_verifications SET used_at = ? WHERE user_id = ? AND used_at IS NULL").bind(nowISO(), user.id).run();
		const token = randomToken(32);
		await env.DB.prepare(
			"INSERT INTO email_verifications (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		)
			.bind(crypto.randomUUID(), user.id, tokenHash(token), new Date(Date.now() + 24 * 3600 * 1000).toISOString(), nowISO())
			.run();
		c.executionCtx.waitUntil(sendVerificationEmail(env, user.email, token));
	}

	return ok(c, { message: "If an account exists for this email, a verification link will be sent" });
});

// ---- POST /auth/forgot-password --------------------------------------

authRoutes.post("/forgot-password", rateLimit("forgot-password"), async (c) => {
	let body: { email?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const email = (body.email ?? "").trim();
	if (!isValidEmail(email)) {
		return error(c, 400, "VALIDATION_ERROR", "invalid email format");
	}

	const env = c.env;
	const user = await getUserByEmail(env, email);
	if (user) {
		await env.DB.prepare("UPDATE password_resets SET used_at = ? WHERE user_id = ? AND used_at IS NULL").bind(nowISO(), user.id).run();
		const token = randomToken(32);
		await env.DB.prepare(
			"INSERT INTO password_resets (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)",
		)
			.bind(crypto.randomUUID(), user.id, tokenHash(token), new Date(Date.now() + 3600 * 1000).toISOString(), nowISO())
			.run();
		c.executionCtx.waitUntil(sendPasswordResetEmail(env, user.email, token));
	}

	return ok(c, { message: "If an account exists for this email, a password reset link will be sent" });
});

// ---- POST /auth/reset-password ---------------------------------------

authRoutes.post("/reset-password", rateLimit("reset-password"), async (c) => {
	let body: { token?: string; password?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const token = body.token ?? "";
	const password = body.password ?? "";
	if (!token || !password) {
		return error(c, 400, "VALIDATION_ERROR", "token and password are required");
	}

	const env = c.env;
	const hash = tokenHash(token);
	const pr = (await env.DB.prepare(
		"SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM password_resets WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?",
	)
		.bind(hash, nowISO())
		.first()) as { user_id: string } | null;
	if (!pr) {
		return error(c, 400, "INVALID_TOKEN", "Invalid or expired reset token");
	}

	const passwordError = validatePassword(password);
	if (passwordError) {
		return error(c, 400, "INVALID_PASSWORD", passwordError);
	}

	const user = await getUserByID(env, pr.user_id);
	if (!user) {
		return error(c, 400, "INVALID_TOKEN", "Invalid or expired reset token");
	}

	const passwordHash = await hashPassword(password, intEnv(env.PBKDF2_ITERATIONS, 310000));
	await env.DB.prepare("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?").bind(passwordHash, nowISO(), user.id).run();
	await env.DB.prepare("UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL").bind(nowISO(), user.id).run();
	await env.DB.prepare("UPDATE password_resets SET used_at = ? WHERE user_id = ? AND used_at IS NULL").bind(nowISO(), user.id).run();
	await audit(env, user.id, "password_reset", clientIP(c), userAgent(c));
	c.executionCtx.waitUntil(sendSecurityNotification(env, user.email, "Your Ilm Nafi password was reset. If this was not you, contact support."));
	c.executionCtx.waitUntil(
		createNotification(
			env,
			user.id,
			"security",
			"Password reset",
			"Your account password was reset. All sessions have been signed out.",
		),
	);

	return ok(c, { message: "Password has been reset successfully" });
});

// ---- GET /auth/me -----------------------------------------------------

authRoutes.get("/me", authRequired, async (c) => {
	const user = await getUserByID(c.env, c.get("userId"));
	if (!user) {
		return error(c, 404, "USER_NOT_FOUND", "User not found");
	}
	const profile = await getProfile(c.env, user.id);
	return ok(c, userResponse(user, profile?.name ?? ""));
});

// ---- GET /auth/sessions ----------------------------------------------

authRoutes.get("/sessions", authRequired, async (c) => {
	const sessions = (await c.env.DB.prepare(
		`SELECT ${SESSION_COLS} FROM sessions WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?`,
	)
		.bind(c.get("userId"), nowISO())
		.all()) as unknown as { results: SessionRow[] };
	return ok(
		c,
		sessions.results.map((s) => ({
			id: s.id,
			device_name: s.device_name,
			device_type: s.device_type,
			ip_address: s.ip_address,
			created_at: s.created_at,
			expires_at: s.expires_at,
			current: false,
		})),
	);
});

// ---- DELETE /auth/delete-account -------------------------------------

authRoutes.delete("/delete-account", authRequired, async (c) => {
	const userId = c.get("userId");
	await c.env.DB.prepare("UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL").bind(nowISO(), userId).run();
	await c.env.DB.prepare("DELETE FROM user_profiles WHERE user_id = ?").bind(userId).run();
	await c.env.DB.prepare("UPDATE users SET deleted_at = ? WHERE id = ?").bind(nowISO(), userId).run();
	await audit(c.env, userId, "account_deleted", clientIP(c), userAgent(c));
	return ok(c, undefined);
});
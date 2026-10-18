import { Hono } from "hono";
import type { AppEnv, Env } from "../lib/env";
import { nowISO, validateName } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { clientIP, userAgent } from "../lib/request";
import { error, ok } from "../lib/response";

export const userRoutes = new Hono<AppEnv>();

userRoutes.use("*", authRequired);

interface UserRow {
	id: string;
	email: string;
	email_verified: number;
	status: string;
	last_login_at: string | null;
	created_at: string;
}

async function getUser(env: Env, id: string): Promise<UserRow | null> {
	return (await env.DB.prepare(
		"SELECT id, email, email_verified, status, last_login_at, created_at FROM users WHERE id = ? AND deleted_at IS NULL",
	)
		.bind(id)
		.first()) as UserRow | null;
}

async function getProfile(env: Env, userId: string): Promise<{ id: string; user_id: string; name: string; bio: string | null; avatar_url: string | null; created_at: string; updated_at: string } | null> {
	return (await env.DB.prepare(
		"SELECT id, user_id, name, bio, avatar_url, created_at, updated_at FROM user_profiles WHERE user_id = ?",
	)
		.bind(userId)
		.first()) as unknown as { id: string; user_id: string; name: string; bio: string | null; avatar_url: string | null; created_at: string; updated_at: string } | null;
}

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

// GET /user/me
userRoutes.get("/me", async (c) => {
	const user = await getUser(c.env, c.get("userId"));
	if (!user) return error(c, 404, "USER_NOT_FOUND", "User not found");
	const profile = await getProfile(c.env, user.id);
	return ok(c, userResponse(user, profile?.name ?? ""));
});

// PUT /user/profile
userRoutes.put("/profile", async (c) => {
	let body: { name?: string; bio?: string; avatar_url?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const name = (body.name ?? "").trim();
	const nameError = validateName(name);
	if (nameError) return error(c, 400, "VALIDATION_ERROR", nameError);

	const userId = c.get("userId");
	const user = await getUser(c.env, userId);
	if (!user) return error(c, 404, "USER_NOT_FOUND", "User not found");
	const profile = await getProfile(c.env, userId);
	if (!profile) return error(c, 404, "PROFILE_NOT_FOUND", "Profile not found");

	await c.env.DB.prepare("UPDATE user_profiles SET name = ?, bio = ?, avatar_url = ?, updated_at = ? WHERE user_id = ?")
		.bind(name, body.bio ?? "", body.avatar_url ?? "", nowISO(), userId)
		.run();

	return ok(c, userResponse(user, name));
});

// DELETE /user/delete-account
userRoutes.delete("/delete-account", async (c) => {
	const userId = c.get("userId");
	await c.env.DB.prepare("UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL").bind(nowISO(), userId).run();
	await c.env.DB.prepare("DELETE FROM user_profiles WHERE user_id = ?").bind(userId).run();
	await c.env.DB.prepare("UPDATE users SET deleted_at = ? WHERE id = ?").bind(nowISO(), userId).run();
	await c.env.DB.prepare(
		"INSERT INTO audit_events (id, user_id, event_type, ip_address, user_agent, created_at) VALUES (?, ?, ?, ?, ?, ?)",
	)
		.bind(crypto.randomUUID(), userId, "account_deleted", clientIP(c), userAgent(c), nowISO())
		.run();
	return ok(c, undefined);
});

// GET /user/sessions
userRoutes.get("/sessions", async (c) => {
	const sessions = (await c.env.DB.prepare(
		"SELECT id, device_name, device_type, ip_address, created_at, expires_at FROM sessions WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?",
	)
		.bind(c.get("userId"), nowISO())
		.all()) as unknown as { results: { id: string; device_name: string; device_type: string; ip_address: string; created_at: string; expires_at: string }[] };
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
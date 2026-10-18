import { Hono } from "hono";
import type { AppEnv, Env } from "../lib/env";
import { nowISO } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { error, ok } from "../lib/response";
import { sendPush } from "../lib/webpush";

export const notificationRoutes = new Hono<AppEnv>();

/** System helper: create a notification and push it to the user's devices. */
export async function createNotification(
	env: Env,
	userId: string,
	type: string,
	title: string,
	body: string,
	data?: unknown,
): Promise<void> {
	await env.DB.prepare(
		"INSERT INTO notifications (user_id, type, title, body, data, created_at) VALUES (?, ?, ?, ?, ?, ?)",
	)
		.bind(userId, type, title, body, data ? JSON.stringify(data) : null, nowISO())
		.run();

	// Web push to subscribed devices (if VAPID is configured).
	if (env.VAPID_PUBLIC_KEY && env.VAPID_PRIVATE_JWK) {
		try {
			const subs = (await env.DB.prepare(
				"SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?",
			)
				.bind(userId)
				.all()) as unknown as { results: { endpoint: string; p256dh: string; auth: string }[] };
			const payload = JSON.stringify({ type, title, body, data: data ?? null });
			for (const sub of subs.results) {
				try {
					await sendPush(env, sub, payload);
				} catch (e) {
					console.error("push send failed for subscription", e);
				}
			}
		} catch (e) {
			console.error("push preparation failed", e);
		}
	}
}

notificationRoutes.use("*", authRequired);

// GET /notifications?limit=
notificationRoutes.get("/", async (c) => {
	const limitParam = parseInt(c.req.query("limit") ?? "", 10);
	const limit = Number.isFinite(limitParam) && limitParam > 0 && limitParam <= 100 ? limitParam : 50;
	const res = await c.env.DB.prepare(
		"SELECT id, type, title, body, data, read_at, created_at FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
	)
		.bind(c.get("userId"), limit)
		.all();
	const items = (res.results as Record<string, unknown>[]).map((n) => ({
		...n,
		read: n.read_at !== null,
		data: n.data ? JSON.parse(String(n.data)) : undefined,
	}));
	const unread = (await c.env.DB.prepare(
		"SELECT COUNT(*) as n FROM notifications WHERE user_id = ? AND read_at IS NULL",
	)
		.bind(c.get("userId"))
		.first()) as { n: number } | null;
	return ok(c, { notifications: items, unread_count: unread?.n ?? 0 });
});

// POST /notifications/:id/read
notificationRoutes.post("/:id/read", async (c) => {
	const id = parseInt(c.req.param("id"), 10);
	if (!Number.isFinite(id) || id <= 0) return error(c, 400, "VALIDATION_ERROR", "Invalid notification id");
	await c.env.DB.prepare("UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ?")
		.bind(nowISO(), id, c.get("userId"))
		.run();
	return ok(c, { status: "ok" });
});

// POST /notifications/read-all
notificationRoutes.post("/read-all", async (c) => {
	await c.env.DB.prepare("UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL")
		.bind(nowISO(), c.get("userId"))
		.run();
	return ok(c, { status: "ok" });
});

// ---- web push subscriptions ------------------------------------------------

// POST /notifications/push/subscribe
notificationRoutes.post("/push/subscribe", async (c) => {
	let body: { endpoint?: string; keys?: { p256dh?: string; auth?: string } };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	const endpoint = body.endpoint ?? "";
	const p256dh = body.keys?.p256dh ?? "";
	const auth = body.keys?.auth ?? "";
	if (!endpoint.startsWith("https://") || !p256dh || !auth) {
		return error(c, 400, "INVALID_REQUEST", "Valid endpoint and keys are required");
	}
	await c.env.DB.prepare(
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (endpoint) DO UPDATE SET
		   user_id = excluded.user_id, p256dh = excluded.p256dh, auth = excluded.auth, last_used_at = excluded.created_at`,
	)
		.bind(c.get("userId"), endpoint, p256dh, auth, nowISO())
		.run();
	return ok(c, { status: "ok" });
});

// POST /notifications/push/unsubscribe
notificationRoutes.post("/push/unsubscribe", async (c) => {
	let body: { endpoint?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	if (!body.endpoint) return error(c, 400, "INVALID_REQUEST", "endpoint is required");
	await c.env.DB.prepare("DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?")
		.bind(c.get("userId"), body.endpoint)
		.run();
	return ok(c, { status: "deleted" });
});

// GET /notifications/push/subscriptions
notificationRoutes.get("/push/subscriptions", async (c) => {
	const res = await c.env.DB.prepare(
		"SELECT id, endpoint, created_at, last_used_at FROM push_subscriptions WHERE user_id = ?",
	)
		.bind(c.get("userId"))
		.all();
	return ok(c, res.results);
});
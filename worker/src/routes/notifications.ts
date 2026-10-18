import { Hono } from "hono";
import type { AppEnv, Env } from "../lib/env";
import { nowISO } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { error, ok } from "../lib/response";

export const notificationRoutes = new Hono<AppEnv>();

/** System helper: create a notification for a user (fire and forget). */
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
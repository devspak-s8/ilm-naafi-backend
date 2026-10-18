import { Hono } from "hono";
import type { AppEnv } from "../lib/env";
import { nowISO } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { error, ok } from "../lib/response";

export const settingsRoutes = new Hono<AppEnv>();

settingsRoutes.use("*", authRequired);

// GET /user/settings
settingsRoutes.get("/settings", async (c) => {
	const row = (await c.env.DB.prepare("SELECT settings FROM user_settings WHERE user_id = ?")
		.bind(c.get("userId"))
		.first()) as { settings: string } | null;
	if (!row) return ok(c, {});
	try {
		return ok(c, JSON.parse(row.settings));
	} catch {
		return ok(c, {});
	}
});

// PUT /user/settings — merge into stored settings
settingsRoutes.put("/settings", async (c) => {
	let body: Record<string, unknown>;
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid request body");
	}
	if (typeof body !== "object" || body === null || Array.isArray(body)) {
		return error(c, 400, "INVALID_REQUEST", "Invalid settings object");
	}

	const userId = c.get("userId");
	const existingRow = (await c.env.DB.prepare("SELECT settings FROM user_settings WHERE user_id = ?")
		.bind(userId)
		.first()) as { settings: string } | null;
	let merged: Record<string, unknown> = {};
	if (existingRow) {
		try {
			merged = JSON.parse(existingRow.settings);
		} catch {
			merged = {};
		}
	}
	merged = { ...merged, ...body };

	const settings = JSON.stringify(merged);
	await c.env.DB.prepare(
		`INSERT INTO user_settings (user_id, settings, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET settings = excluded.settings, updated_at = excluded.updated_at`,
	)
		.bind(userId, settings, nowISO())
		.run();
	return ok(c, merged);
});
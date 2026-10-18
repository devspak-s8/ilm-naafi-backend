import { Hono } from "hono";
import type { AppEnv } from "../lib/env";
import { nowISO, todayUTC } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { created, error, ok } from "../lib/response";

export const adhkarRoutes = new Hono<AppEnv>();

const CATEGORY_COLS = "id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at";
const DHIKR_COLS = "id, category_id, title, arabic_text, transliteration, translation, repeat_count, sort_order, audio_url, is_active, created_at, updated_at";
const SOURCE_COLS = "id, dhikr_id, source_type, collection, reference, source_text, verification_status, created_at";

const SLUG_MAP: Record<string, string> = {
	morning: "Morning",
	evening: "Evening",
	"post-salah": "Post-Salah",
	"before-sleep": "Before Sleeping",
	"upon-waking": "Upon Waking",
};

function parseID(value: string): number | null {
	const n = parseInt(value, 10);
	return Number.isFinite(n) ? n : null;
}

// GET /adhkar/categories
adhkarRoutes.get("/categories", async (c) => {
	const res = await c.env.DB.prepare(`SELECT ${CATEGORY_COLS} FROM dhikr_categories WHERE is_active = 1 ORDER BY sort_order ASC`).all();
	return ok(c, res.results);
});

// GET /adhkar/categories/:id
adhkarRoutes.get("/categories/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_CATEGORY_ID", "Invalid category id");
	const category = await c.env.DB.prepare(`SELECT ${CATEGORY_COLS} FROM dhikr_categories WHERE id = ?`).bind(id).first();
	if (!category) return error(c, 404, "NOT_FOUND", "ADHKAR_CATEGORY_NOT_FOUND");
	return ok(c, category);
});

// GET /adhkar/categories/:id/adhkar
adhkarRoutes.get("/categories/:id/adhkar", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_CATEGORY_ID", "Invalid category id");
	const category = await c.env.DB.prepare("SELECT id FROM dhikr_categories WHERE id = ?").bind(id).first();
	if (!category) return error(c, 404, "NOT_FOUND", "ADHKAR_CATEGORY_NOT_FOUND");
	const res = await c.env.DB.prepare(`SELECT ${DHIKR_COLS} FROM adhkar WHERE category_id = ? AND is_active = 1 ORDER BY sort_order ASC`).bind(id).all();
	return ok(c, res.results);
});

// GET /adhkar/:slug  (registered before /:id for parity with the Go router)
adhkarRoutes.get("/:slug", async (c) => {
	const slug = c.req.param("slug");
	if (!slug) return error(c, 400, "VALIDATION_ERROR", "INVALID_SLUG");
	const name = SLUG_MAP[slug.toLowerCase()] ?? slug;
	const category = await c.env.DB.prepare("SELECT id FROM dhikr_categories WHERE name = ? AND is_active = 1 LIMIT 1").bind(name).first();
	if (!category) return error(c, 404, "NOT_FOUND", "ADHKAR_NOT_FOUND");
	const res = await c.env.DB.prepare(`SELECT ${DHIKR_COLS} FROM adhkar WHERE category_id = ? AND is_active = 1 ORDER BY sort_order ASC`)
		.bind((category as { id: number }).id)
		.all();
	return ok(c, res.results);
});

// GET /adhkar/:id  (dhikr with sources; parity fallback)
adhkarRoutes.get("/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_DHIKR_ID", "Invalid dhikr id");
	const dhikr = await c.env.DB.prepare(`SELECT ${DHIKR_COLS} FROM adhkar WHERE id = ?`).bind(id).first();
	if (!dhikr) return error(c, 404, "NOT_FOUND", "ADHKAR_NOT_FOUND");
	const sources = await c.env.DB.prepare(`SELECT ${SOURCE_COLS} FROM adhkar_sources WHERE dhikr_id = ?`).bind(id).all();
	return ok(c, { dhikr, sources: sources.results });
});

// ---- authenticated adhkar routes --------------------------------------

const adhkarProtected = new Hono<AppEnv>();
adhkarProtected.use("*", authRequired);

// POST /adhkar/:id/complete
adhkarProtected.post("/:id/complete", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_DHIKR_ID", "Invalid dhikr id");
	let increment = 1;
	try {
		const body = (await c.req.json()) as { increment?: number };
		if (body.increment && body.increment > 0) increment = body.increment;
	} catch {
		// body optional
	}

	const dhikr = (await c.env.DB.prepare(`SELECT ${DHIKR_COLS} FROM adhkar WHERE id = ?`).bind(id).first()) as { repeat_count: number } | null;
	if (!dhikr) return error(c, 404, "NOT_FOUND", "ADHKAR_NOT_FOUND");

	const userId = c.get("userId");
	const date = todayUTC();
	const required = dhikr.repeat_count > 0 ? dhikr.repeat_count : 1;

	const existing = (await c.env.DB.prepare(
		"SELECT completed_count FROM adhkar_completion WHERE user_id = ? AND dhikr_id = ? AND completion_date = ?",
	)
		.bind(userId, id, date)
		.first()) as { completed_count: number } | null;

	let completedCount = increment;
	if (existing) completedCount = existing.completed_count + increment;
	const reached = completedCount >= required;
	if (reached) completedCount = required;
	const completedAt = reached ? new Date().toISOString() : null;

	await c.env.DB.prepare(
		`INSERT INTO adhkar_completion (user_id, dhikr_id, completion_date, completed_count, required_count, completed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, dhikr_id, completion_date) DO UPDATE SET
		   completed_count = excluded.completed_count,
		   required_count = excluded.required_count,
		   completed_at = excluded.completed_at`,
	)
		.bind(userId, id, date, completedCount, required, completedAt, nowISO())
		.run();

	return ok(c, {
		user_id: userId,
		dhikr_id: id,
		completion_date: date,
		completed_count: completedCount,
		required_count: required,
		completed_at: completedAt,
	});
});

// GET /adhkar/progress/today?date=
adhkarProtected.get("/progress/today", async (c) => {
	const userId = c.get("userId");
	const date = c.req.query("date") || todayUTC();
	const progress: Record<string, unknown> = { date };

	const catName = async (name: string): Promise<number> => {
		const row = (await c.env.DB.prepare("SELECT id FROM dhikr_categories WHERE name = ? AND is_active = 1 LIMIT 1").bind(name).first()) as { id: number } | null;
		return row?.id ?? 0;
	};
	const fullyCompleted = async (categoryID: number): Promise<boolean> => {
		if (categoryID === 0) return false;
		const total = (await c.env.DB.prepare("SELECT COUNT(*) as n FROM adhkar WHERE category_id = ? AND is_active = 1").bind(categoryID).first()) as { n: number };
		if (!total || total.n === 0) return false;
		const done = (await c.env.DB.prepare(
			`SELECT COUNT(DISTINCT ac.dhikr_id) as n
			 FROM adhkar_completion ac JOIN adhkar a ON ac.dhikr_id = a.id
			 WHERE ac.user_id = ? AND a.category_id = ? AND ac.completion_date = ? AND ac.completed_count >= ac.required_count`,
		)
			.bind(userId, categoryID, date)
			.first()) as { n: number };
		return (done?.n ?? 0) >= total.n;
	};

	progress.morning_completed = await fullyCompleted(await catName("Morning"));
	progress.evening_completed = await fullyCompleted(await catName("Evening"));
	progress.post_salah_completed = await fullyCompleted(await catName("Post-Salah"));

	const total = (await c.env.DB.prepare("SELECT COUNT(*) as n FROM adhkar WHERE is_active = 1").first()) as { n: number };
	if (total && total.n > 0) {
		const done = (await c.env.DB.prepare(
			`SELECT COUNT(DISTINCT ac.dhikr_id) as n
			 FROM adhkar_completion ac
			 WHERE ac.user_id = ? AND ac.completion_date = ? AND ac.completed_count >= ac.required_count`,
		)
			.bind(userId, date)
			.first()) as { n: number };
		progress.overall_percent = Math.floor(((done?.n ?? 0) * 100) / total.n);
	} else {
		progress.overall_percent = 0;
	}

	return ok(c, progress);
});

// GET /adhkar/progress/history?days=
adhkarProtected.get("/progress/history", async (c) => {
	const userId = c.get("userId");
	let days = parseInt(c.req.query("days") ?? "", 10);
	if (!Number.isFinite(days) || days <= 0) days = 7;
	const history: unknown[] = [];
	for (let i = 0; i < days; i++) {
		const d = new Date();
		d.setUTCDate(d.getUTCDate() - i);
		const date = d.toISOString().slice(0, 10);
		const res = await c.env.DB.prepare(
			`SELECT id, user_id, dhikr_id, completion_date, completed_count, required_count, completed_at, created_at
			 FROM adhkar_completion WHERE user_id = ? AND completion_date = ?`,
		)
			.bind(userId, date)
			.all();
		history.push(...res.results);
	}
	return ok(c, history);
});

// POST /adhkar/reminders
adhkarProtected.post("/reminders", async (c) => {
	const userId = c.get("userId");
	let body: { category_id?: number; enabled?: boolean; time?: string; notification_type?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid body");
	}
	const categoryID = body.category_id ?? 0;
	if (categoryID <= 0) return error(c, 400, "VALIDATION_ERROR", "INVALID_REMINDER_REQUEST");
	const category = await c.env.DB.prepare("SELECT id FROM dhikr_categories WHERE id = ?").bind(categoryID).first();
	if (!category) return error(c, 404, "NOT_FOUND", "ADHKAR_CATEGORY_NOT_FOUND");
	const notifType = body.notification_type || "normal";
	await c.env.DB.prepare(
		`INSERT INTO adhkar_reminders (user_id, category_id, enabled, time, notification_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, category_id) DO UPDATE SET
		   enabled = excluded.enabled, time = excluded.time,
		   notification_type = excluded.notification_type, updated_at = excluded.updated_at`,
	)
		.bind(userId, categoryID, body.enabled === false ? 0 : 1, body.time ?? "", notifType, nowISO(), nowISO())
		.run();
	return created(c, { status: "ok" });
});

// PUT /adhkar/reminders/:id
adhkarProtected.put("/reminders/:id", async (c) => {
	const userId = c.get("userId");
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_CATEGORY_ID", "Invalid category id");
	let body: { enabled?: boolean; time?: string; notification_type?: string };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Invalid body");
	}
	await c.env.DB.prepare(
		"UPDATE adhkar_reminders SET enabled = ?, time = ?, notification_type = ?, updated_at = ? WHERE user_id = ? AND category_id = ?",
	)
		.bind(body.enabled === false ? 0 : 1, body.time ?? "", body.notification_type || "normal", nowISO(), userId, id)
		.run();
	return ok(c, { status: "ok" });
});

// GET /adhkar/reminders
adhkarProtected.get("/reminders", async (c) => {
	const res = await c.env.DB.prepare(
		"SELECT id, user_id, category_id, enabled, time, notification_type, created_at, updated_at FROM adhkar_reminders WHERE user_id = ?",
	)
		.bind(c.get("userId"))
		.all();
	return ok(c, res.results);
});

// DELETE /adhkar/reminders/:id
adhkarProtected.delete("/reminders/:id", async (c) => {
	const userId = c.get("userId");
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_CATEGORY_ID", "Invalid category id");
	await c.env.DB.prepare("DELETE FROM adhkar_reminders WHERE user_id = ? AND category_id = ?").bind(userId, id).run();
	return ok(c, { status: "deleted" });
});

adhkarRoutes.route("/", adhkarProtected);
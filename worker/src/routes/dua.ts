import { Hono } from "hono";
import type { AppEnv } from "../lib/env";
import { error, ok } from "../lib/response";

export const duaRoutes = new Hono<AppEnv>();

const CATEGORY_COLS = "id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at";
const DUA_COLS = "id, category_id, title, arabic_text, transliteration, translation, source_id, audio_url, sort_order, is_active, created_at, updated_at";
const SOURCE_COLS = "id, collection, reference, source_text, verification_status";

const SLUG_MAP: Record<string, string> = {
	protection: "Protection",
	forgiveness: "Forgiveness",
	guidance: "Guidance",
	family: "Family",
	travel: "Travel",
	health: "Health",
	rizq: "Rizq",
	patience: "Patience",
	general: "General",
};

function parseID(value: string): number | null {
	const n = parseInt(value, 10);
	return Number.isFinite(n) ? n : null;
}

// GET /dua/categories
duaRoutes.get("/categories", async (c) => {
	const res = await c.env.DB.prepare(`SELECT ${CATEGORY_COLS} FROM dhikr_categories WHERE is_active = 1 ORDER BY sort_order ASC`).all();
	return ok(c, res.results);
});

// GET /dua/categories/:id/duas
duaRoutes.get("/categories/:id/duas", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_CATEGORY_ID", "Invalid category id");
	const category = await c.env.DB.prepare("SELECT id FROM dhikr_categories WHERE id = ?").bind(id).first();
	if (!category) return error(c, 404, "NOT_FOUND", "DUA_CATEGORY_NOT_FOUND");
	const res = await c.env.DB.prepare(`SELECT ${DUA_COLS} FROM duas WHERE category_id = ? AND is_active = 1 ORDER BY sort_order ASC`).bind(id).all();
	return ok(c, res.results);
});

// GET /dua/:id
duaRoutes.get("/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_DUA_ID", "Invalid dua id");
	const dua = await c.env.DB.prepare(`SELECT ${DUA_COLS} FROM duas WHERE id = ?`).bind(id).first();
	if (!dua) return error(c, 404, "NOT_FOUND", "DUA_NOT_FOUND");
	const sources = (dua as { source_id: number | null }).source_id
		? await c.env.DB.prepare(`SELECT ${SOURCE_COLS} FROM adhkar_sources WHERE id = ?`).bind((dua as { source_id: number }).source_id).all()
		: { results: [] };
	return ok(c, { dua, sources: sources.results });
});

// GET /dua/slug/:slug
duaRoutes.get("/slug/:slug", async (c) => {
	const slug = c.req.param("slug");
	if (!slug) return error(c, 400, "VALIDATION_ERROR", "INVALID_SLUG");
	const name = SLUG_MAP[slug.toLowerCase()] ?? slug.toLowerCase();
	const res = await c.env.DB.prepare(
		`SELECT d.id, d.category_id, d.title, d.arabic_text, d.transliteration, d.translation, d.source_id, d.audio_url, d.sort_order, d.is_active, d.created_at, d.updated_at
		 FROM duas d JOIN dhikr_categories c ON d.category_id = c.id
		 WHERE c.name = ? AND d.is_active = 1 ORDER BY d.sort_order ASC`,
	).bind(name).all();
	if (res.results.length === 0) return error(c, 404, "NOT_FOUND", "DUA_NOT_FOUND");
	return ok(c, res.results);
});
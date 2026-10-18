import { Hono } from "hono";
import type { AppEnv } from "../lib/env";
import { notFound, validation } from "../lib/errors";
import { nowISO } from "../lib/security";
import { authRequired } from "../middleware/auth";
import { created, error, handleError, ok } from "../lib/response";

export const quranRoutes = new Hono<AppEnv>();

const SURAH_COLS = "id, number, name, arabic_name, english_name, revelation_type, ayah_count, page_number, created_at, updated_at";
const AYAH_COLS = "id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at";
const RECITER_COLS = "id, name, arabic_name, identifier, description, audio_base_url, is_active, created_at, updated_at";
const AUDIO_COLS = "id, reciter_id, surah_id, ayah_id, audio_url, duration_ms, start_ms, end_ms, format, created_at";

function rowToBool(value: unknown): boolean {
	return value === 1 || value === true || value === "1";
}

function ayahRow(r: Record<string, unknown>) {
	return {
		id: r.id,
		surah_id: r.surah_id,
		ayah_number: r.ayah_number,
		text: r.text,
		page_number: r.page_number ?? undefined,
		juz_number: r.juz_number ?? undefined,
		hizb_number: r.hizb_number ?? undefined,
		rub_el_hizb: r.rub_el_hizb ?? undefined,
		sajdah: rowToBool(r.sajdah),
		created_at: r.created_at,
	};
}

function parseID(value: string): number | null {
	const n = parseInt(value, 10);
	return Number.isFinite(n) ? n : null;
}

// GET /quran/surahs
quranRoutes.get("/surahs", async (c) => {
	const res = await c.env.DB.prepare(`SELECT ${SURAH_COLS} FROM quran_surahs ORDER BY number ASC`).all();
	return ok(c, res.results);
});

// GET /quran/surahs/:id
quranRoutes.get("/surahs/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_SURAH_ID", "Invalid surah id");
	const surah = await c.env.DB.prepare(`SELECT ${SURAH_COLS} FROM quran_surahs WHERE id = ?`).bind(id).first();
	if (!surah) return error(c, 404, "NOT_FOUND", "SURAH_NOT_FOUND");
	const ayahs = await c.env.DB.prepare(`SELECT ${AYAH_COLS} FROM quran_ayahs WHERE surah_id = ? ORDER BY ayah_number ASC`).bind(id).all();
	return ok(c, { surah, ayahs: ayahs.results.map(ayahRow) });
});

// GET /quran/surahs/:id/ayahs
quranRoutes.get("/surahs/:id/ayahs", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_SURAH_ID", "Invalid surah id");
	const surah = await c.env.DB.prepare("SELECT id FROM quran_surahs WHERE id = ?").bind(id).first();
	if (!surah) return error(c, 404, "NOT_FOUND", "SURAH_NOT_FOUND");
	const ayahs = await c.env.DB.prepare(`SELECT ${AYAH_COLS} FROM quran_ayahs WHERE surah_id = ? ORDER BY ayah_number ASC`).bind(id).all();
	return ok(c, ayahs.results.map(ayahRow));
});

// GET /quran/ayahs/:id
quranRoutes.get("/ayahs/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_AYAH_ID", "Invalid ayah id");
	const ayah = await c.env.DB.prepare(`SELECT ${AYAH_COLS} FROM quran_ayahs WHERE id = ?`).bind(id).first();
	if (!ayah) return error(c, 404, "NOT_FOUND", "AYAH_NOT_FOUND");
	return ok(c, ayahRow(ayah as unknown as Record<string, unknown>));
});

// GET /quran/juz/:id
quranRoutes.get("/juz/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_JUZ_ID", "Invalid juz id");
	if (id <= 0 || id > 30) return error(c, 400, "VALIDATION_ERROR", "INVALID_JUZ");
	const ayahs = await c.env.DB.prepare(`SELECT ${AYAH_COLS} FROM quran_ayahs WHERE juz_number = ? ORDER BY surah_id, ayah_number ASC`).bind(id).all();
	if (ayahs.results.length === 0) return error(c, 404, "NOT_FOUND", "JUZ_NOT_FOUND");
	return ok(c, ayahs.results.map(ayahRow));
});

// GET /quran/search?q=&limit=
quranRoutes.get("/search", async (c) => {
	const query = (c.req.query("q") ?? "").trim();
	if (!query) return error(c, 400, "VALIDATION_ERROR", "INVALID_QUERY");
	let limit = parseInt(c.req.query("limit") ?? "", 10);
	if (!Number.isFinite(limit) || limit <= 0 || limit > 200) limit = 50;
	const results = await c.env.DB.prepare(
		`SELECT qa.surah_id, qs.name as surah_name, qa.ayah_number, qa.id as ayah_id, qa.text
		 FROM quran_ayahs qa JOIN quran_surahs qs ON qa.surah_id = qs.id
		 WHERE qa.text LIKE ?
		 ORDER BY qa.surah_id, qa.ayah_number
		 LIMIT ?`,
	)
		.bind(`%${query}%`, limit)
		.all();
	return ok(c, results.results);
});

// GET /quran/reciters
quranRoutes.get("/reciters", async (c) => {
	const res = await c.env.DB.prepare(`SELECT ${RECITER_COLS} FROM quran_reciters WHERE is_active = 1 ORDER BY name ASC`).all();
	return ok(c, res.results);
});

// GET /quran/reciters/:id
quranRoutes.get("/reciters/:id", async (c) => {
	const id = parseID(c.req.param("id"));
	if (id === null) return error(c, 400, "INVALID_RECITER_ID", "Invalid reciter id");
	const reciter = await c.env.DB.prepare(`SELECT ${RECITER_COLS} FROM quran_reciters WHERE id = ?`).bind(id).first();
	if (!reciter) return error(c, 404, "NOT_FOUND", "RECITER_NOT_FOUND");
	return ok(c, reciter);
});

// GET /quran/audio/surah/:surahID?reciter=
quranRoutes.get("/audio/surah/:surahID", async (c) => {
	const surahID = parseID(c.req.param("surahID"));
	const reciterID = parseInt(c.req.query("reciter") ?? "", 10);
	if (surahID === null) return error(c, 400, "INVALID_SURAH_ID", "Invalid surah id");
	if (!Number.isFinite(reciterID) || reciterID <= 0) return error(c, 400, "INVALID_RECITER_ID", "Invalid reciter id");
	const res = await c.env.DB.prepare(`SELECT ${AUDIO_COLS} FROM quran_audio WHERE surah_id = ? AND reciter_id = ? ORDER BY ayah_id ASC`)
		.bind(surahID, reciterID)
		.all();
	return ok(c, res.results);
});

// GET /quran/audio/ayah/:ayahID?reciter=
quranRoutes.get("/audio/ayah/:ayahID", async (c) => {
	const ayahID = parseID(c.req.param("ayahID"));
	const reciterID = parseInt(c.req.query("reciter") ?? "", 10);
	if (ayahID === null) return error(c, 400, "INVALID_AYAH_ID", "Invalid ayah id");
	if (!Number.isFinite(reciterID) || reciterID <= 0) return error(c, 400, "INVALID_RECITER_ID", "Invalid reciter id");
	const audio = await c.env.DB.prepare(`SELECT ${AUDIO_COLS} FROM quran_audio WHERE ayah_id = ? AND reciter_id = ?`)
		.bind(ayahID, reciterID)
		.first();
	if (!audio) return error(c, 404, "NOT_FOUND", "AUDIO_NOT_FOUND");
	return ok(c, audio);
});

// ---- authenticated quran routes --------------------------------------

const quranProtected = new Hono<AppEnv>();
quranProtected.use("*", authRequired);

// GET /quran/bookmarks
quranProtected.get("/bookmarks", async (c) => {
	const res = await c.env.DB.prepare("SELECT id, user_id, ayah_id, created_at FROM quran_bookmarks WHERE user_id = ? ORDER BY created_at DESC")
		.bind(c.get("userId"))
		.all();
	return ok(c, res.results);
});

// POST /quran/bookmarks
quranProtected.post("/bookmarks", async (c) => {
	let body: { ayah_id?: number };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Valid ayah_id is required");
	}
	const ayahID = body.ayah_id ?? 0;
	if (ayahID <= 0) return error(c, 400, "INVALID_REQUEST", "Valid ayah_id is required");
	const ayah = await c.env.DB.prepare("SELECT id FROM quran_ayahs WHERE id = ?").bind(ayahID).first();
	if (!ayah) return error(c, 404, "NOT_FOUND", "AYAH_NOT_FOUND");
	await c.env.DB.prepare(
		"INSERT INTO quran_bookmarks (user_id, ayah_id, created_at) VALUES (?, ?, ?) ON CONFLICT (user_id, ayah_id) DO NOTHING",
	)
		.bind(c.get("userId"), ayahID, nowISO())
		.run();
	return created(c, { status: "ok" });
});

// DELETE /quran/bookmarks/:ayahID
quranProtected.delete("/bookmarks/:ayahID", async (c) => {
	const ayahID = parseID(c.req.param("ayahID"));
	if (ayahID === null) return error(c, 400, "INVALID_AYAH_ID", "Invalid ayah id");
	await c.env.DB.prepare("DELETE FROM quran_bookmarks WHERE user_id = ? AND ayah_id = ?").bind(c.get("userId"), ayahID).run();
	return ok(c, { status: "deleted" });
});

// GET /quran/progress
quranProtected.get("/progress", async (c) => {
	const progress = await c.env.DB.prepare(
		"SELECT id, user_id, last_surah_id, last_ayah_id, last_page_number, updated_at FROM quran_reading_progress WHERE user_id = ?",
	)
		.bind(c.get("userId"))
		.first();
	if (!progress) return ok(c, {});
	return ok(c, progress);
});

// PUT /quran/progress
quranProtected.put("/progress", async (c) => {
	let body: { surah_id?: number; ayah_id?: number; page_number?: number | null };
	try {
		body = await c.req.json();
	} catch {
		return error(c, 400, "INVALID_REQUEST", "Valid surah_id and ayah_id are required");
	}
	const surahID = body.surah_id ?? 0;
	const ayahID = body.ayah_id ?? 0;
	if (surahID <= 0 || ayahID <= 0) {
		return error(c, 400, "INVALID_REQUEST", "Valid surah_id and ayah_id are required");
	}
	const ayah = await c.env.DB.prepare("SELECT id FROM quran_ayahs WHERE surah_id = ? AND ayah_number = ?").bind(surahID, ayahID).first();
	if (!ayah) return error(c, 400, "VALIDATION_ERROR", "INVALID_READING_POSITION");

	await c.env.DB.prepare(
		`INSERT INTO quran_reading_progress (user_id, last_surah_id, last_ayah_id, last_page_number, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET
		   last_surah_id = excluded.last_surah_id,
		   last_ayah_id = excluded.last_ayah_id,
		   last_page_number = excluded.last_page_number,
		   updated_at = excluded.updated_at`,
	)
		.bind(c.get("userId"), surahID, ayahID, body.page_number ?? null, nowISO())
		.run();
	await c.env.DB.prepare("INSERT INTO quran_reading_history (user_id, surah_id, ayah_id, started_at) VALUES (?, ?, ?, ?)")
		.bind(c.get("userId"), surahID, ayahID, nowISO())
		.run();
	return ok(c, { status: "ok" });
});

// GET /quran/continue
quranProtected.get("/continue", async (c) => {
	const progress = await c.env.DB.prepare(
		"SELECT id, user_id, last_surah_id, last_ayah_id, last_page_number, updated_at FROM quran_reading_progress WHERE user_id = ?",
	)
		.bind(c.get("userId"))
		.first();
	if (!progress) return ok(c, {});
	return ok(c, progress);
});

quranRoutes.route("/", quranProtected);
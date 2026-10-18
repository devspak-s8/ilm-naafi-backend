import { Hono } from "hono";
import type { Context } from "hono";
import type { AppEnv } from "../lib/env";
import { newAppError } from "../lib/errors";
import { error, handleError, ok } from "../lib/response";

/**
 * Hadith content backed by the Sunnah.com API (https://sunnah.com).
 * An optional SUNNAH_API_KEY secret is sent as x-api-key when set.
 */
export const hadithRoutes = new Hono<AppEnv>();

const BASE = "https://api.sunnah.com/v1";

function hadithStatusError(status: number) {
	switch (status) {
		case 400:
		case 422:
			return newAppError(400, "HADITH_INVALID_REQUEST", "HADITH_INVALID_REQUEST");
		case 401:
		case 403:
			return newAppError(502, "HADITH_API_KEY_REQUIRED", "Sunnah.com API key required — set the SUNNAH_API_KEY secret");
		case 404:
			return newAppError(404, "HADITH_NOT_FOUND", "Hadith not found");
		case 429:
			return newAppError(429, "HADITH_PROVIDER_RATE_LIMITED", "HADITH_PROVIDER_RATE_LIMITED");
		default:
			return newAppError(503, "HADITH_PROVIDER_UNAVAILABLE", "Hadith provider is unavailable");
	}
}

async function sunnahGet(c: Context<AppEnv>, path: string): Promise<unknown> {
	const headers: Record<string, string> = { accept: "application/json" };
	const key = (c.env as AppEnv["Bindings"] & { SUNNAH_API_KEY?: string }).SUNNAH_API_KEY;
	if (key) headers["x-api-key"] = key;
	const res = await fetch(`${BASE}${path}`, { headers });
	if (!res.ok) throw hadithStatusError(res.status);
	return res.json();
}

// GET /hadith/collections
hadithRoutes.get("/collections", async (c) => {
	try {
		const data = await sunnahGet(c, "/collections?limit=50");
		return ok(c, (data as { data: unknown[] }).data ?? []);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /hadith/collections/:collectionName/books
hadithRoutes.get("/collections/:collectionName/books", async (c) => {
	const name = c.req.param("collectionName");
	if (!name) return error(c, 400, "INVALID_COLLECTION", "Invalid collection");
	try {
		const data = await sunnahGet(c, `/collections/${encodeURIComponent(name)}/books?limit=100`);
		return ok(c, (data as { data: unknown[] }).data ?? []);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /hadith/collections/:collectionName/books/:bookNumber/hadiths?page=&limit=
hadithRoutes.get("/collections/:collectionName/books/:bookNumber/hadiths", async (c) => {
	const name = c.req.param("collectionName");
	const book = c.req.param("bookNumber");
	if (!name || !book) return error(c, 400, "INVALID_COLLECTION", "Invalid collection or book");
	const page = parseInt(c.req.query("page") ?? "", 10);
	const limit = parseInt(c.req.query("limit") ?? "", 10);
	const pageParam = Number.isFinite(page) && page > 0 ? page : 1;
	const limitParam = Math.min(Math.max(Number.isFinite(limit) && limit > 0 ? limit : 20, 1), 50);
	try {
		const data = await sunnahGet(c, `/collections/${encodeURIComponent(name)}/books/${encodeURIComponent(book)}/hadiths?page=${pageParam}&limit=${limitParam}`);
		return ok(c, data);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /hadith/:collectionName/:hadithNumber
hadithRoutes.get("/:collectionName/:hadithNumber", async (c) => {
	const name = c.req.param("collectionName");
	const number = c.req.param("hadithNumber");
	if (!name || !number) return error(c, 400, "INVALID_HADITH_REFERENCE", "Invalid hadith reference");
	try {
		const data = await sunnahGet(c, `/hadiths/${encodeURIComponent(name)}/${encodeURIComponent(number)}`);
		return ok(c, data);
	} catch (e) {
		return handleError(c, e);
	}
});
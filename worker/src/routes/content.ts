import { Hono } from "hono";
import type { Context } from "hono";
import type { AppEnv, Env } from "../lib/env";
import { intEnv } from "../lib/env";
import { newAppError } from "../lib/errors";
import { error, handleError, ok } from "../lib/response";

// ---- provider types -----------------------------------------------------

export interface VerseQuery {
	language: string;
	words: boolean;
	translations: string;
	tafsirs: string;
	audio: number;
	page: number;
	perPage: number;
}

export interface SearchQuery {
	query: string;
	page: number;
	size: number;
	translationIds: string;
}

// ---- provider -----------------------------------------------------------

const PROVIDER = "quran_foundation";

interface ProviderConfig {
	apiBaseURL: string;
	oauthBaseURL: string;
	clientID: string;
	clientSecret: string;
	timeoutSeconds: number;
}

let tokenCache: { token: string; expiresAt: number } | null = null;

function providerConfig(env: Env): ProviderConfig {
	const production = env.QURAN_FOUNDATION_ENV === "production";
	return {
		apiBaseURL: (env.QURAN_FOUNDATION_API_BASE_URL || (production ? "https://apis.quran.foundation" : "https://apis-prelive.quran.foundation")).replace(/\/+$/, ""),
		oauthBaseURL: (env.QURAN_FOUNDATION_OAUTH_BASE_URL || (production ? "https://oauth2.quran.foundation" : "https://prelive-oauth2.quran.foundation")).replace(/\/+$/, ""),
		clientID: env.QURAN_FOUNDATION_CLIENT_ID ?? "",
		clientSecret: env.QURAN_FOUNDATION_CLIENT_SECRET ?? "",
		timeoutSeconds: Math.max(1, intEnv(env.QURAN_PROVIDER_TIMEOUT, 15)),
	};
}

async function fetchWithTimeout(url: string, init: RequestInit, timeoutSeconds: number): Promise<Response> {
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort(), timeoutSeconds * 1000);
	try {
		return await fetch(url, { ...init, signal: controller.signal });
	} finally {
		clearTimeout(timer);
	}
}

async function oauthToken(cfg: ProviderConfig): Promise<string> {
	if (tokenCache && tokenCache.expiresAt > Date.now()) return tokenCache.token;
	if (!cfg.clientID || !cfg.clientSecret) {
		throw newAppError(503, "QURAN_PROVIDER_UNAVAILABLE", "Quran Foundation credentials are not configured");
	}
	const body = new URLSearchParams({ grant_type: "client_credentials", scope: "content search" });
	const res = await fetchWithTimeout(
		`${cfg.oauthBaseURL}/oauth2/token`,
		{
			method: "POST",
			headers: {
				"content-type": "application/x-www-form-urlencoded",
				authorization: `Basic ${btoa(`${cfg.clientID}:${cfg.clientSecret}`)}`,
			},
			body: body.toString(),
		},
		cfg.timeoutSeconds,
	);
	if (!res.ok) throw providerStatusError(res.status);
	const data = (await res.json()) as { access_token?: string; expires_in?: number };
	if (!data.access_token) throw newAppError(502, "QURAN_PROVIDER_INVALID_RESPONSE", "token response did not contain access_token");
	const expires = data.expires_in && data.expires_in > 0 ? data.expires_in : 3600;
	tokenCache = { token: data.access_token, expiresAt: Date.now() + (expires - 60) * 1000 };
	return data.access_token;
}

function providerStatusError(status: number) {
	switch (status) {
		case 400:
		case 422:
			return newAppError(400, "QURAN_INVALID_REQUEST", "QURAN_INVALID_REQUEST");
		case 401:
			return newAppError(502, "QURAN_PROVIDER_UNAUTHORIZED", "QURAN_PROVIDER_UNAUTHORIZED");
		case 403:
			return newAppError(502, "QURAN_PROVIDER_FORBIDDEN", "QURAN_PROVIDER_FORBIDDEN");
		case 404:
			return newAppError(404, "QURAN_RESOURCE_NOT_FOUND", "Quran resource not found");
		case 429:
			return newAppError(429, "QURAN_PROVIDER_RATE_LIMITED", "QURAN_PROVIDER_RATE_LIMITED");
		default:
			return newAppError(503, "QURAN_PROVIDER_UNAVAILABLE", "Quran provider is unavailable");
	}
}

async function providerGet(env: Env, endpoint: string, params: Record<string, string>, authRetry = true): Promise<unknown> {
	if (env.QURAN_PROVIDER !== PROVIDER) {
		throw newAppError(503, "QURAN_PROVIDER_UNAVAILABLE", `unsupported Quran provider "${env.QURAN_PROVIDER}"`);
	}
	const cfg = providerConfig(env);
	const token = await oauthToken(cfg);
	const qs = new URLSearchParams(params).toString();
	const url = `${cfg.apiBaseURL}${endpoint}${qs ? `?${qs}` : ""}`;
	let res = await fetchWithTimeout(
		url,
		{
			headers: {
				"x-auth-token": token,
				"x-client-id": cfg.clientID,
				accept: "application/json",
			},
		},
		cfg.timeoutSeconds,
	);
	if (res.status === 401 && authRetry) {
		tokenCache = null;
		return providerGet(env, endpoint, params, false);
	}
	if (res.status === 429 || res.status >= 500) {
		await new Promise((r) => setTimeout(r, 1000));
		res = await fetchWithTimeout(
			url,
			{
				headers: {
					"x-auth-token": token,
					"x-client-id": cfg.clientID,
					accept: "application/json",
				},
			},
			cfg.timeoutSeconds,
		);
	}
	if (!res.ok) throw providerStatusError(res.status);
	return res.json();
}

// ---- routes --------------------------------------------------------------

export const contentRoutes = new Hono<AppEnv>();

function parseVerseQuery(c: Context<AppEnv>): VerseQuery {
	const page = parseInt(c.req.query("page") ?? "", 10);
	const perPage = parseInt(c.req.query("per_page") ?? "", 10);
	const audio = parseInt(c.req.query("audio") ?? "", 10);
	return {
		language: c.req.query("language") ?? "",
		words: (c.req.query("words") ?? "") === "true",
		translations: c.req.query("translations") ?? "",
		tafsirs: c.req.query("tafsirs") ?? "",
		audio: Number.isFinite(audio) ? audio : 0,
		page: Number.isFinite(page) ? page : 0,
		perPage: Number.isFinite(perPage) ? perPage : 0,
	};
}

function normalizeVerseQuery(q: VerseQuery): VerseQuery {
	return {
		...q,
		language: q.language || "en",
		page: q.page > 0 ? q.page : 1,
		perPage: Math.min(Math.max(q.perPage > 0 ? q.perPage : 20, 1), 50),
	};
}

function verseParams(q: VerseQuery): Record<string, string> {
	const params: Record<string, string> = {
		language: q.language || "en",
		words: String(q.words),
		fields: "text_uthmani,text_imlaei",
	};
	if (q.translations) params.translations = q.translations;
	if (q.tafsirs) params.tafsirs = q.tafsirs;
	if (q.audio > 0) params.audio = String(q.audio);
	if (q.page > 0) params.page = String(q.page);
	if (q.perPage > 0) params.per_page = String(q.perPage);
	return params;
}

function positiveParam(value: string): number | null {
	const n = parseInt(value, 10);
	return Number.isFinite(n) && n > 0 ? n : null;
}

// GET /api/v1/quran/chapters
contentRoutes.get("/chapters", async (c) => {
	try {
		const data = await providerGet(c.env, "/content/api/v4/chapters", { language: c.req.query("language") || "en" });
		const chapters = (data as { chapters: unknown[] }).chapters ?? [];
		return ok(c, chapters.map((ch) => ({ ...(ch as object), provider: PROVIDER })));
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/chapters/:chapter
contentRoutes.get("/chapters/:chapter", async (c) => {
	const id = positiveParam(c.req.param("chapter"));
	if (id === null || id > 114) return error(c, 400, "INVALID_CHAPTER", "Invalid chapter");
	try {
		const data = await providerGet(c.env, `/content/api/v4/chapters/${id}`, { language: c.req.query("language") || "en" });
		const chapter = (data as { chapter: object }).chapter;
		return ok(c, { ...chapter, provider: PROVIDER });
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/chapters/:chapter/verses
contentRoutes.get("/chapters/:chapter/verses", async (c) => {
	const id = positiveParam(c.req.param("chapter"));
	if (id === null || id > 114) return error(c, 400, "INVALID_CHAPTER", "Invalid chapter");
	try {
		const q = normalizeVerseQuery(parseVerseQuery(c));
		const data = await providerGet(c.env, `/content/api/v4/verses/by_chapter/${id}`, verseParams(q));
		return ok(c, data);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/verses/:verseKey
contentRoutes.get("/verses/:verseKey", async (c) => {
	const key = c.req.param("verseKey").trim();
	if (!/^[1-9][0-9]*:[1-9][0-9]*$/.test(key)) return error(c, 400, "INVALID_VERSE_KEY", "Invalid verse key");
	try {
		const q = normalizeVerseQuery(parseVerseQuery(c));
		const data = await providerGet(c.env, `/content/api/v4/verses/by_key/${encodeURIComponent(key)}`, verseParams(q));
		return ok(c, (data as { verse: object }).verse);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/juz/:juz
contentRoutes.get("/juz/:juz", async (c) => {
	const id = positiveParam(c.req.param("juz"));
	if (id === null || id > 30) return error(c, 400, "INVALID_JUZ", "Invalid juz");
	const mushaf = parseInt(c.req.query("mushaf_id") ?? "", 10);
	try {
		const params: Record<string, string> = {};
		if (Number.isFinite(mushaf) && mushaf > 0) params.mushaf = String(mushaf);
		const data = await providerGet(c.env, `/content/api/v4/juzs/${id}`, params);
		return ok(c, { ...((data as { juz: object }).juz ?? {}), provider: PROVIDER });
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/pages/:page
contentRoutes.get("/pages/:page", async (c) => {
	const id = positiveParam(c.req.param("page"));
	if (id === null || id > 1000) return error(c, 400, "INVALID_PAGE", "Invalid page");
	try {
		const q = normalizeVerseQuery(parseVerseQuery(c));
		const data = await providerGet(c.env, `/content/api/v4/verses/by_page/${id}`, verseParams(q));
		return ok(c, data);
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/search
contentRoutes.get("/search", async (c) => {
	const query = (c.req.query("q") ?? "").trim();
	if (!query) return error(c, 400, "VALIDATION_ERROR", "INVALID_QUERY");
	let page = parseInt(c.req.query("page") ?? "", 10);
	if (!Number.isFinite(page) || page <= 0) page = 1;
	let size = parseInt(c.req.query("size") ?? "", 10);
	if (!Number.isFinite(size) || size <= 0) size = 20;
	if (size > 50) return error(c, 400, "VALIDATION_ERROR", "SEARCH_LIMIT_TOO_LARGE");
	const params: Record<string, string> = {
		mode: "advanced",
		query,
		page: String(page),
		size: String(size),
		highlight: "0",
	};
	const translationIds = c.req.query("translation_ids") ?? "";
	if (translationIds) params.translation_ids = translationIds;
	try {
		const data = (await providerGet(c.env, "/api/v1/search", params)) as { result?: object; pagination?: object };
		return ok(c, { ...(data.result ?? {}), pagination: data.pagination ?? (data.result as { pagination?: object } | undefined)?.pagination });
	} catch (e) {
		return handleError(c, e);
	}
});

// GET /api/v1/quran/resources/:resource
contentRoutes.get("/resources/:resource", async (c) => {
	const resourceType = c.req.param("resource").toLowerCase();
	if (!["translations", "tafsirs", "recitations"].includes(resourceType)) {
		return error(c, 400, "VALIDATION_ERROR", "INVALID_RESOURCE_TYPE");
	}
	try {
		const data = await providerGet(c.env, `/content/api/v4/resources/${encodeURIComponent(resourceType)}`, {});
		const resources = (data as { resources: object[] }).resources ?? [];
		return ok(c, resources.map((r) => ({ ...r, resource_type: resourceType, provider: PROVIDER })));
	} catch (e) {
		return handleError(c, e);
	}
});
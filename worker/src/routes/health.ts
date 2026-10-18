import { Hono } from "hono";
import type { AppEnv } from "../lib/env";
import { ok } from "../lib/response";

export const healthRoutes = new Hono<AppEnv>();

// GET /health
healthRoutes.get("/health", async (c) => {
	const status: Record<string, unknown> = {
		status: "healthy",
		timestamp: new Date().toISOString(),
		version: "1.0.0",
	};
	try {
		await c.env.DB.prepare("SELECT 1").first();
		status.database = "healthy";
		return c.json(status, 200);
	} catch {
		status.status = "unhealthy";
		status.database = "unhealthy";
		status.error = "database connection failed";
		return c.json(status, 503);
	}
});

// GET /ready
healthRoutes.get("/ready", async (c) => {
	const status: Record<string, unknown> = {
		status: "ready",
		timestamp: new Date().toISOString(),
	};
	try {
		// A real table touch proves both connectivity and that migrations ran.
		const row = (await c.env.DB.prepare("SELECT COUNT(*) as n FROM users").first()) as { n: number } | null;
		status.database = "ready";
		status.user_count = row?.n ?? 0;
		return c.json(status, 200);
	} catch {
		status.status = "not ready";
		status.database = "not ready";
		return c.json(status, 503);
	}
});
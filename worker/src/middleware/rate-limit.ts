import type { MiddlewareHandler } from "hono";
import type { AppEnv } from "../lib/env";
import { intEnv } from "../lib/env";
import { clientIP } from "../lib/request";
import { error } from "../lib/response";

/**
 * Per-route KV-backed rate limiter. Degrades gracefully (no KV binding =
 * no limiting). Keys are per-route-per-IP with the configured window.
 */
export function rateLimit(route: string): MiddlewareHandler<AppEnv> {
	return async (c, next) => {
		const kv = c.env.RATE_LIMIT;
		if (!kv) return next();
		const limit = intEnv(c.env.RATE_LIMIT_REQUESTS, 10);
		const window = Math.max(1, intEnv(c.env.RATE_LIMIT_WINDOW, 60));
		const key = `rl:${route}:${clientIP(c)}`;
		const current = parseInt((await kv.get(key)) ?? "0", 10) || 0;
		if (current >= limit) {
			c.header("Retry-After", String(window));
			return error(c, 429, "RATE_LIMIT_EXCEEDED", "Too many requests");
		}
		await kv.put(key, String(current + 1), { expirationTtl: window });
		await next();
	};
}
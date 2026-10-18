import type { Context } from "hono";

/** Real client IP: Workers always run behind Cloudflare, so CF-Connecting-IP is authoritative. */
export function clientIP(c: Context): string {
	const cf = c.req.header("CF-Connecting-IP");
	if (cf) return cf;
	const xff = c.req.header("X-Forwarded-For");
	if (xff) return xff.split(",")[0].trim();
	const xrip = c.req.header("X-Real-IP");
	if (xrip) return xrip;
	return "unknown";
}

export function userAgent(c: Context): string {
	return c.req.header("User-Agent") ?? "";
}

export function origin(c: Context): string {
	return c.req.header("Origin") ?? "";
}
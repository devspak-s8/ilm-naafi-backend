import type { MiddlewareHandler } from "hono";
import type { AppEnv } from "../lib/env";
import { verifyAccessToken } from "../lib/jwt";
import { error } from "../lib/response";

/** Requires a valid Bearer access token; sets userId/sessionId variables. */
export const authRequired: MiddlewareHandler<AppEnv> = async (c, next) => {
	const header = c.req.header("Authorization");
	if (!header) {
		return error(c, 401, "MISSING_TOKEN", "Missing authorization header");
	}
	const parts = header.split(" ");
	if (parts.length !== 2 || parts[0] !== "Bearer") {
		return error(c, 401, "INVALID_TOKEN", "Invalid authorization header");
	}
	const claims = await verifyAccessToken(c.env.JWT_ACCESS_SECRET, parts[1]);
	if (!claims) {
		return error(c, 401, "INVALID_TOKEN", "Invalid or expired token");
	}
	c.set("userId", claims.sub);
	c.set("sessionId", claims.session_id);
	await next();
};
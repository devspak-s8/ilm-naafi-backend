import { Hono } from "hono";
import type { MiddlewareHandler } from "hono";
import type { AppEnv } from "./lib/env";
import { error, handleError } from "./lib/response";
import { clientIP } from "./lib/request";
import { authRoutes } from "./routes/auth";
import { userRoutes } from "./routes/user";
import { quranRoutes } from "./routes/quran";
import { contentRoutes } from "./routes/content";
import { adhkarRoutes } from "./routes/adhkar";
import { duaRoutes } from "./routes/dua";
import { hadithRoutes } from "./routes/hadith";
import { healthRoutes } from "./routes/health";
import { notificationRoutes } from "./routes/notifications";
import { passkeyRoutes } from "./routes/passkeys";
import { settingsRoutes } from "./routes/settings";

const app = new Hono<AppEnv>();

// ---- global middleware -------------------------------------------------

const securityHeaders: MiddlewareHandler<AppEnv> = async (c, next) => {
	await next();
	c.header("X-Content-Type-Options", "nosniff");
	c.header("X-Frame-Options", "DENY");
	c.header("X-XSS-Protection", "1; mode=block");
	c.header("Strict-Transport-Security", "max-age=31536000; includeSubDomains");
	c.header("Content-Security-Policy", "default-src 'self'");
};

const logger: MiddlewareHandler<AppEnv> = async (c, next) => {
	const start = Date.now();
	await next();
	const ray = c.req.header("CF-Ray");
	if (ray) {
		console.log(`${c.req.method} ${c.req.path} ${clientIP(c)} cf-ray=${ray} ${Date.now() - start}ms`);
	} else {
		console.log(`${c.req.method} ${c.req.path} ${clientIP(c)} ${Date.now() - start}ms`);
	}
};

const cors: MiddlewareHandler<AppEnv> = async (c, next) => {
	const origin = c.req.header("Origin");
	const allowed = (c.env.CORS_ALLOWED_ORIGINS ?? "")
		.split(",")
		.map((o) => o.trim())
		.filter(Boolean);
	if (origin && allowed.includes(origin)) {
		c.header("Access-Control-Allow-Origin", origin);
		c.header("Vary", "Origin");
		c.header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");
		c.header("Access-Control-Allow-Headers", "Authorization, Content-Type");
		c.header("Access-Control-Max-Age", "86400");
		if (c.req.method === "OPTIONS") {
			return c.body(null, 204);
		}
	}
	await next();
};

app.use("*", logger);
app.use("*", securityHeaders);
app.use("*", cors);

// ---- routes --------------------------------------------------------------

app.route("/", healthRoutes);
app.route("/auth", authRoutes);
app.route("/auth/passkeys", passkeyRoutes);
app.route("/user", userRoutes);
app.route("/user", settingsRoutes);
app.route("/notifications", notificationRoutes);
app.route("/quran", quranRoutes);
app.route("/api/v1/quran", contentRoutes);
app.route("/adhkar", adhkarRoutes);
app.route("/dua", duaRoutes);
app.route("/hadith", hadithRoutes);

app.notFound((c) => error(c, 404, "NOT_FOUND", "Not found"));

app.onError((err, c) => {
	console.error("worker error", err);
	return handleError(c, err);
});

export default app;
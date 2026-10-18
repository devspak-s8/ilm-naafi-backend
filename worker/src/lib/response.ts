import type { Context } from "hono";
import { AppError } from "./errors";

/** Standard API envelope: { success, data } or { success, error: { code, message } }. */
export function ok(c: Context, data: unknown): Response {
	return c.json({ success: true, data }, 200);
}

export function created(c: Context, data: unknown): Response {
	return c.json({ success: true, data }, 201);
}

export function error(c: Context, status: number, code: string, message: string): Response {
	return c.json({ success: false, error: { code, message } }, status as 200);
}

export function handleError(c: Context, e: unknown): Response {
	if (e instanceof AppError) {
		return error(c, e.status, e.code, e.message);
	}
	console.error("unhandled error", e);
	return error(c, 500, "INTERNAL_ERROR", "An error occurred");
}
const encoder = new TextEncoder();
const decoder = new TextDecoder();

// Cloudflare Workers' WebCrypto caps PBKDF2 at 100,000 iterations.
export const MAX_PBKDF2_ITERATIONS = 100_000;

export function toB64Url(data: Uint8Array | ArrayBuffer): string {
	const bytes = data instanceof Uint8Array ? data : new Uint8Array(data);
	let bin = "";
	for (const b of bytes) bin += String.fromCharCode(b);
	return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function fromB64Url(value: string): Uint8Array<ArrayBuffer> {
	const b64 = value.replace(/-/g, "+").replace(/_/g, "/");
	const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
	const bin = atob(padded);
	const out = new Uint8Array(bin.length);
	for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
	return out;
}

/** Cryptographically secure random token (base64url), matches Go's GenerateSecureToken. */
export function randomToken(byteLength = 32): string {
	const buf = new Uint8Array(byteLength);
	crypto.getRandomValues(buf);
	return toB64Url(buf);
}

/** Hex encoding of the token string, matching Go's GenerateSecureHash. */
export function tokenHash(token: string): string {
	const bytes = encoder.encode(token);
	let hex = "";
	for (const b of bytes) hex += b.toString(16).padStart(2, "0");
	return hex;
}

async function pbkdf2(password: string, salt: Uint8Array, iterations: number): Promise<Uint8Array> {
	const keyMaterial = await crypto.subtle.importKey("raw", encoder.encode(password), "PBKDF2", false, [
		"deriveBits",
	]);
	const bits = await crypto.subtle.deriveBits(
		{ name: "PBKDF2", salt: salt as BufferSource, iterations, hash: "SHA-256" },
		keyMaterial,
		256,
	);
	return new Uint8Array(bits);
}

/** PBKDF2-HMAC-SHA256, stored as pbkdf2-sha256$<iterations>$<salt>$<hash>. */
export async function hashPassword(password: string, iterations: number): Promise<string> {
	const clamped = Math.min(Math.max(iterations, 10_000), MAX_PBKDF2_ITERATIONS);
	const salt = crypto.getRandomValues(new Uint8Array(16));
	const key = await pbkdf2(password, salt, clamped);
	return `pbkdf2-sha256$${clamped}$${toB64Url(salt)}$${toB64Url(key)}`;
}

/**
 * Verifies a PBKDF2 hash. Legacy Go argon2id hashes ($argon2id$...) cannot be
 * verified in Workers and always return false; affected users must use the
 * password reset flow.
 */
export async function verifyPassword(password: string, stored: string): Promise<boolean> {
	const parts = stored.split("$");
	if (parts[0] !== "pbkdf2-sha256" || parts.length !== 4) return false;
	const iterations = Math.min(parseInt(parts[1], 10), MAX_PBKDF2_ITERATIONS);
	if (!Number.isFinite(iterations) || iterations <= 0) return false;
	const salt = fromB64Url(parts[2]);
	const expected = fromB64Url(parts[3]);
	const actual = await pbkdf2(password, salt, iterations);
	return timingSafeEqual(actual, expected);
}

function timingSafeEqual(a: Uint8Array, b: Uint8Array): boolean {
	if (a.length !== b.length) return false;
	let diff = 0;
	for (let i = 0; i < a.length; i++) diff |= a[i] ^ b[i];
	return diff === 0;
}

export function isValidEmail(email: string): boolean {
	return email.length > 0 && email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

/** Returns an error message or null. Mirrors internal/security/validator.go. */
export function validatePassword(password: string): string | null {
	if (password.length < 8) return "password must be at least 8 characters";
	if (password.length > 128) return "password must not exceed 128 characters";
	if (!/[A-Z]/.test(password)) return "password must contain at least one uppercase letter";
	if (!/[a-z]/.test(password)) return "password must contain at least one lowercase letter";
	if (!/[0-9]/.test(password)) return "password must contain at least one number";
	if (!/[^A-Za-z0-9]/.test(password)) return "password must contain at least one special character";
	return null;
}

/** Returns an error message or null. */
export function validateName(name: string): string | null {
	if (name.length < 2) return "name must be at least 2 characters";
	if (name.length > 100) return "name must not exceed 100 characters";
	if (!/^[a-zA-Z\s'-]{2,100}$/.test(name)) return "name contains invalid characters";
	return null;
}

export function nowISO(): string {
	return new Date().toISOString();
}

export function todayUTC(): string {
	return new Date().toISOString().slice(0, 10);
}

export function decodeUtf8(data: Uint8Array): string {
	return decoder.decode(data);
}

export function encodeUtf8(value: string): Uint8Array {
	return encoder.encode(value);
}
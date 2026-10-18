import { encodeUtf8, fromB64Url, toB64Url } from "./security";

export interface AccessClaims {
	sub: string;
	session_id: string;
	iss: string;
	iat: number;
	exp: number;
}

async function hmacKey(secret: string, usage: "sign" | "verify"): Promise<CryptoKey> {
	return crypto.subtle.importKey(
		"raw",
		encodeUtf8(secret),
		{ name: "HMAC", hash: "SHA-256" },
		false,
		[usage],
	);
}

export async function signAccessToken(secret: string, claims: AccessClaims): Promise<string> {
	const header = toB64Url(encodeUtf8(JSON.stringify({ alg: "HS256", typ: "JWT" })));
	const payload = toB64Url(encodeUtf8(JSON.stringify(claims)));
	const data = `${header}.${payload}`;
	const key = await hmacKey(secret, "sign");
	const sig = await crypto.subtle.sign("HMAC", key, encodeUtf8(data));
	return `${data}.${toB64Url(sig)}`;
}

export async function verifyAccessToken(secret: string, token: string): Promise<AccessClaims | null> {
	const parts = token.split(".");
	if (parts.length !== 3) return null;
	if (!parts[0] || !parts[1] || !parts[2]) return null;

	const key = await hmacKey(secret, "verify");
	let valid = false;
	try {
		valid = await crypto.subtle.verify(
			"HMAC",
			key,
			fromB64Url(parts[2]) as BufferSource,
			encodeUtf8(`${parts[0]}.${parts[1]}`),
		);
	} catch {
		return null;
	}
	if (!valid) return null;

	let claims: unknown;
	try {
		claims = JSON.parse(new TextDecoder().decode(fromB64Url(parts[1])));
	} catch {
		return null;
	}
	if (typeof claims !== "object" || claims === null) return null;
	const c = claims as Record<string, unknown>;
	if (typeof c.sub !== "string" || typeof c.session_id !== "string") return null;
	if (typeof c.exp !== "number" || c.exp * 1000 < Date.now()) return null;
	return { sub: c.sub, session_id: c.session_id, iss: String(c.iss ?? ""), iat: Number(c.iat ?? 0), exp: c.exp };
}
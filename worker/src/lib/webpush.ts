import { encodeUtf8, fromB64Url, toB64Url } from "./security";

/**
 * Web Push (RFC 8030 / RFC 8291, aes128gcm) — implemented with native
 * WebCrypto: VAPID ES256 signatures, ECDH P-256 key agreement, HKDF-SHA256,
 * AES-128-GCM. No external crypto dependencies.
 */

export interface PushSubscriptionRecord {
	endpoint: string;
	p256dh: string; // base64url of the raw uncompressed P-256 point (65 bytes)
	auth: string; // base64url, 16 bytes
}

export interface VapidKeys {
	publicKey: string; // base64url raw uncompressed point
	privateJwk: JsonWebKey; // ECDSA P-256 private key as JWK
}

function b64UrlToBytes(s: string): Uint8Array<ArrayBuffer> {
	return fromB64Url(s);
}

function concatBytes(...parts: Uint8Array[]): Uint8Array {
	const total = parts.reduce((n, p) => n + p.length, 0);
	const out = new Uint8Array(total);
	let offset = 0;
	for (const p of parts) {
		out.set(p, offset);
		offset += p.length;
	}
	return out;
}

async function hmacSha256(keyBytes: Uint8Array, data: Uint8Array): Promise<Uint8Array<ArrayBuffer>> {
	const key = await crypto.subtle.importKey("raw", keyBytes as BufferSource, { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
	const sig = await crypto.subtle.sign("HMAC", key, data as BufferSource);
	return new Uint8Array(sig);
}

/** HKDF-Expand (RFC 5869) with SHA-256. */
async function hkdfExpand(prk: Uint8Array, info: Uint8Array, length: number): Promise<Uint8Array> {
	const blocks: Uint8Array[] = [];
	let previous = new Uint8Array(0);
	let counter = 0;
	while (blocks.reduce((n, b) => n + b.length, 0) < length) {
		counter += 1;
		const input = concatBytes(previous, info, Uint8Array.of(counter));
		previous = await hmacSha256(prk, input);
		blocks.push(previous);
	}
	const out = concatBytes(...blocks);
	return new Uint8Array(out.slice(0, length));
}

type KeyUsage = "decrypt" | "deriveBits" | "deriveKey" | "encrypt" | "generateKey" | "importKey" | "sign" | "unwrapKey" | "verify" | "wrapKey";

async function importEcdsaKey(jwk: JsonWebKey, usage: "sign" | "deriveBits"): Promise<CryptoKey> {
	const usages: KeyUsage[] = usage === "sign" ? ["sign"] : ["deriveBits"];
	return crypto.subtle.importKey("jwk", jwk, { name: "ECDSA", namedCurve: "P-256" }, false, usages);
}

/** Builds and signs the VAPID JWT. */
async function vapidJwt(privateJwk: JsonWebKey, audience: string, subject: string): Promise<string> {
	const header = toB64Url(encodeUtf8(JSON.stringify({ typ: "JWT", alg: "ES256" })));
	const payload = toB64Url(
		encodeUtf8(
			JSON.stringify({
				aud: audience,
				exp: Math.floor(Date.now() / 1000) + 12 * 3600,
				sub: subject,
			}),
		),
	);
	const data = `${header}.${payload}`;
	const key = await importEcdsaKey(privateJwk, "sign");
	const sig = await crypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, key, encodeUtf8(data));
	return `${data}.${toB64Url(sig)}`;
}

/** Sends an encrypted push message to the subscription endpoint. */
export async function sendPush(env: Env, sub: PushSubscriptionRecord, payload: string): Promise<boolean> {
	const publicKey = env.VAPID_PUBLIC_KEY;
	const privateJwk = parseVapidPrivateJwk(env.VAPID_PRIVATE_JWK);
	const audience = new URL(sub.endpoint).origin;
	const subject = `mailto:${env.EMAIL_FROM}`;

	const jwt = await vapidJwt(privateJwk, audience, subject);

	const salt = crypto.getRandomValues(new Uint8Array(16));
	const authSecret = b64UrlToBytes(sub.auth);
	const clientPublic = b64UrlToBytes(sub.p256dh);

	// ECDH: derive shared secret between our P-256 key and the client key.
	const serverPrivate = await importEcdsaKey(privateJwk, "deriveBits");
	const clientKey = await crypto.subtle.importKey(
		"raw",
		clientPublic as BufferSource,
		{ name: "ECDSA", namedCurve: "P-256" },
		false,
		[],
	);
	const ecdhSecret = new Uint8Array(
		await crypto.subtle.deriveBits(
			{ name: "ECDH", public: clientKey } as unknown as SubtleCryptoDeriveKeyAlgorithm,
			serverPrivate,
			256,
		),
	);

	// HKDF: PRK = HMAC(auth_secret, ecdh_secret)
	const prk = await hmacSha256(authSecret, ecdhSecret);
	const ikm = concatBytes(encodeUtf8("WebPush: info"), Uint8Array.of(0), clientPublic, b64UrlToBytes(publicKey));
	const cek = await hkdfExpand(prk, ikm, 16); // AES-128 key
	const nonce = await hkdfExpand(prk, concatBytes(encodeUtf8("Content-Encoding: aes128gcm"), Uint8Array.of(0)), 12);

	// RFC 8188 padding: 0x02 + zeroes, total length multiple of 16.
	const plaintext = encodeUtf8(payload);
	const padLen = 16 - ((plaintext.length + 1) % 16); // +1 for delimiter byte
	const padded = concatBytes(plaintext, Uint8Array.of(2), new Uint8Array(padLen === 16 ? 0 : padLen));

	const aesKey = await crypto.subtle.importKey("raw", cek as BufferSource, { name: "AES-GCM" }, false, ["encrypt"]);
	const ciphertext = new Uint8Array(
		await crypto.subtle.encrypt({ name: "AES-GCM", iv: nonce as BufferSource }, aesKey, padded as BufferSource),
	);

	// Body: salt(16) | rs(4) | keylen(1) | key(65) | ciphertext
	const body = concatBytes(
		salt,
		new Uint8Array([0, 0, 16, 0]), // record size 4096 big-endian
		Uint8Array.of(65),
		b64UrlToBytes(publicKey),
		ciphertext,
	);

	try {
		const res = await fetch(sub.endpoint, {
			method: "POST",
			headers: {
				"content-encoding": "aes128gcm",
				"content-type": "application/octet-stream",
				ttl: "86400",
				authorization: `vapid t=${jwt}, k=${publicKey}`,
			},
			body,
		});
		if (!res.ok) {
			console.error("push send failed", res.status, await res.text().catch(() => ""));
			return false;
		}
		return true;
	} catch (e) {
		console.error("push send error", e);
		return false;
	}
}

export function parseVapidPrivateJwk(raw: string): JsonWebKey {
	try {
		return JSON.parse(raw) as JsonWebKey;
	} catch {
		throw new Error("VAPID_PRIVATE_JWK is not valid JSON");
	}
}

// Local type import to avoid a circular dependency on env.ts types.
type Env = {
	VAPID_PUBLIC_KEY: string;
	VAPID_PRIVATE_JWK: string;
	EMAIL_FROM: string;
};
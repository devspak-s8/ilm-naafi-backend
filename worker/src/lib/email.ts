import type { Env } from "./env";

/**
 * Sends email through the MailChannels transactional API. Requires the
 * sending domain to be verified: add the SPF record for api.mailchannels.net
 * and a DKIM record for the _mailchannels selector (see Cloudflare docs on
 * "Send email from Workers using MailChannels").
 */
export async function sendEmail(env: Env, to: string, subject: string, html: string): Promise<void> {
	const payload = {
		personalizations: [{ to: [{ email: to }] }],
		from: { email: env.EMAIL_FROM, name: env.EMAIL_FROM_NAME },
		subject,
		content: [{ type: "text/html", value: html }],
	};
	try {
		const res = await fetch("https://api.mailchannels.net/tx/v1/send", {
			method: "POST",
			headers: { "content-type": "application/json" },
			body: JSON.stringify(payload),
		});
		if (!res.ok) {
			console.error("MailChannels send failed", res.status, await res.text().catch(() => ""));
		}
	} catch (e) {
		console.error("MailChannels send error", e);
	}
}

export function sendVerificationEmail(env: Env, to: string, token: string): Promise<void> {
	const url = `${env.APP_URL}/auth/verify-email?token=${encodeURIComponent(token)}`;
	const html = `<html><body><p>Hello,</p><p>Please verify your email by clicking the link below:</p><a href="${url}">Verify Email</a><p>This link will expire in 24 hours.</p></body></html>`;
	return sendEmail(env, to, "Verify your email address", html);
}

export function sendPasswordResetEmail(env: Env, to: string, token: string): Promise<void> {
	const url = `${env.APP_URL}/auth/reset-password?token=${encodeURIComponent(token)}`;
	const html = `<html><body><p>Hello,</p><p>You requested a password reset. Click the link below:</p><a href="${url}">Reset Password</a><p>This link will expire in 1 hour.</p></body></html>`;
	return sendEmail(env, to, "Reset your password", html);
}

export function sendSecurityNotification(env: Env, to: string, event: string): Promise<void> {
	const html = `<html><body><p>Hello,</p><p>A security event occurred: ${event}</p></body></html>`;
	return sendEmail(env, to, "Security notification", html);
}
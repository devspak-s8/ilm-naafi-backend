import type { Env } from "./env";

/**
 * Email sending — Cloudflare-first.
 *
 * 1. Cloudflare Email Service (native `send_email` binding; Workers Paid plan
 *    and the sending domain must be onboarded to Email Service).
 *    Falls back silently if the binding is absent or the send is rejected.
 * 2. MailChannels transactional API (requires the MailChannels SPF/DKIM
 *    DNS records on the sending domain).
 */
export async function sendEmail(env: Env, to: string, subject: string, html: string): Promise<void> {
	if (env.SEND_EMAIL) {
		try {
			await env.SEND_EMAIL.send({ to, from: env.EMAIL_FROM, subject, html });
			return;
		} catch (e) {
			console.error("Email Service send failed, falling back to MailChannels", e);
		}
	}
	await sendViaMailChannels(env, to, subject, html);
}

async function sendViaMailChannels(env: Env, to: string, subject: string, html: string): Promise<void> {
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
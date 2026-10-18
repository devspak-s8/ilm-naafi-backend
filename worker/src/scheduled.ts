import type { Env } from "./lib/env";
import { nowISO } from "./lib/security";
import { sendPush } from "./lib/webpush";

/**
 * Adhkar reminder scheduler — runs on a Cron Trigger every 10 minutes.
 *
 * Reminder `time` values are the USER'S LOCAL time. Each user's timezone is
 * read from user_settings (settings JSON, key "timezone", IANA name,
 * default "UTC"). Dedup happens per user-local day via last_reminded_at.
 */
export async function runAdhkarReminders(env: Env): Promise<void> {
	const now = new Date();
	const windowStart = new Date(now.getTime() - 10 * 60 * 1000);

	const rows = (await env.DB.prepare(
		"SELECT id, user_id, category_id, time, last_reminded_at FROM adhkar_reminders WHERE enabled = 1",
	).all()) as unknown as {
		results: { id: number; user_id: string; category_id: number; time: string; last_reminded_at: string | null }[];
	};

	const tzCache = new Map<string, string>();
	let processed = 0;

	for (const rem of rows.results) {
		if (!rem.time || !/^\d{2}:\d{2}$/.test(rem.time)) continue;

		// User timezone from settings
		let tz = tzCache.get(rem.user_id);
		if (tz === undefined) {
			const settingsRow = (await env.DB.prepare("SELECT settings FROM user_settings WHERE user_id = ?")
				.bind(rem.user_id)
				.first()) as { settings: string } | null;
			tz = "UTC";
			if (settingsRow) {
				try {
					const settings = JSON.parse(settingsRow.settings);
					if (typeof settings.timezone === "string" && settings.timezone) {
						tz = settings.timezone;
					}
				} catch {
					// ignore malformed settings
				}
			}
			tzCache.set(rem.user_id, tz as string);
		}
		const timezone: string = tz ?? "UTC";

		const nowLocal = hhmmInTz(now, timezone);
		const windowLocal = hhmmInTz(windowStart, timezone);
		if (nowLocal === null || windowLocal === null) continue;

		const due = windowLocal > nowLocal
			? rem.time <= nowLocal || rem.time > windowLocal // crossed midnight locally
			: rem.time > windowLocal && rem.time <= nowLocal;

		if (!due) continue;

		// Dedup: skip if already reminded today (in the user's local timezone)
		if (rem.last_reminded_at) {
			const lastLocalDay = localDateInTz(new Date(rem.last_reminded_at), timezone);
			const nowLocalDay = localDateInTz(now, timezone);
			if (lastLocalDay && nowLocalDay && lastLocalDay === nowLocalDay) continue;
		}

		const category = (await env.DB.prepare("SELECT name FROM dhikr_categories WHERE id = ?")
			.bind(rem.category_id)
			.first()) as { name: string } | null;
		const categoryName = category?.name ?? "adhkar";

		await env.DB.prepare(
			"INSERT INTO notifications (user_id, type, title, body, data, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		)
			.bind(
				rem.user_id,
				"adhkar_reminder",
				`Time for ${categoryName}`,
				`Your reminder for ${categoryName} is due. Take a moment for dhikr.`,
				JSON.stringify({ category_id: rem.category_id, time: rem.time, timezone: tz }),
				nowISO(),
			)
			.run();

		// Push to the user's devices (never let push failures break the cron)
		if (env.VAPID_PUBLIC_KEY && env.VAPID_PRIVATE_JWK) {
			try {
				const subs = (await env.DB.prepare(
					"SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?",
				)
					.bind(rem.user_id)
					.all()) as unknown as { results: { endpoint: string; p256dh: string; auth: string }[] };
				const payload = JSON.stringify({
					type: "adhkar_reminder",
					title: `Time for ${categoryName}`,
					body: `Your reminder for ${categoryName} is due.`,
					data: { category_id: rem.category_id, time: rem.time },
				});
				for (const sub of subs.results) {
					try {
						await sendPush(env, sub, payload);
					} catch (e) {
						console.error("push send failed for subscription", e);
					}
				}
			} catch (e) {
				console.error("push preparation failed", e);
			}
		}

		await env.DB.prepare("UPDATE adhkar_reminders SET last_reminded_at = ? WHERE id = ?")
			.bind(nowISO(), rem.id)
			.run();

		processed += 1;
	}

	console.log(`adhkar reminders processed: ${processed}`);
}

/** HH:MM in a given IANA timezone, or null if the timezone is invalid. */
function hhmmInTz(date: Date, tz: string): string | null {
	try {
		const parts = new Intl.DateTimeFormat("en-GB", {
			hour: "2-digit",
			minute: "2-digit",
			hour12: false,
			timeZone: tz,
		}).format(date);
		return parts.replace(/^24:/, "00:");
	} catch {
		return null;
	}
}

/** YYYY-MM-DD in a given IANA timezone, or null if invalid. */
function localDateInTz(date: Date, tz: string): string | null {
	try {
		const parts = new Intl.DateTimeFormat("en-CA", {
			year: "numeric",
			month: "2-digit",
			day: "2-digit",
			timeZone: tz,
		}).format(date);
		return parts;
	} catch {
		return null;
	}
}
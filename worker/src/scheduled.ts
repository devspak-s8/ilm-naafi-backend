import type { Env } from "./lib/env";
import { nowISO } from "./lib/security";

/**
 * Adhkar reminder scheduler — runs on a Cron Trigger every 10 minutes.
 *
 * Finds enabled reminders whose `time` (UTC HH:MM) falls inside the cron
 * window and which haven't fired today yet, creates an in-app notification
 * for each, and stamps `last_reminded_at` to prevent duplicates.
 */
export async function runAdhkarReminders(env: Env): Promise<void> {
	const now = new Date();
	const windowStart = new Date(now.getTime() - 10 * 60 * 1000);
	const todayStart = new Date(now.getTime());
	todayStart.setUTCHours(0, 0, 0, 0);

	const hhmmNow = `${String(now.getUTCHours()).padStart(2, "0")}:${String(now.getUTCMinutes()).padStart(2, "0")}`;
	const hhmmWindow = `${String(windowStart.getUTCHours()).padStart(2, "0")}:${String(windowStart.getUTCMinutes()).padStart(2, "0")}`;

	const crossesMidnight = windowStart.getUTCDate() !== now.getUTCDate();
	const timeCondition = crossesMidnight
		? "(time <= ? OR time > ?)"
		: "(time > ? AND time <= ?)";
	const timeParams = crossesMidnight ? [hhmmNow, hhmmWindow] : [hhmmWindow, hhmmNow];

	const rows = (await env.DB.prepare(
		`SELECT id, user_id, category_id, time FROM adhkar_reminders
		 WHERE enabled = 1 AND ${timeCondition}
		   AND (last_reminded_at IS NULL OR last_reminded_at < ?)`,
	)
		.bind(...timeParams, todayStart.toISOString())
		.all()) as unknown as { results: { id: number; user_id: string; category_id: number; time: string }[] };

	for (const rem of rows.results) {
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
				JSON.stringify({ category_id: rem.category_id, time: rem.time }),
				nowISO(),
			)
			.run();

		await env.DB.prepare("UPDATE adhkar_reminders SET last_reminded_at = ? WHERE id = ?")
			.bind(nowISO(), rem.id)
			.run();
	}

	console.log(`adhkar reminders processed: ${rows.results.length}`);
}
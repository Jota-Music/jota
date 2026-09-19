import { DiscordEnabled, SetDiscordEnabled } from "@bindings/app";
import { useSignal } from "@preact/signals";
import { Headphones } from "lucide-preact";
import { useEffect } from "preact/hooks";
import * as media from "@/lib/music/views/stores/media-session";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

export function DiscordSettings() {
	const enabled = useSignal(false);
	const ready = useSignal(false);

	useEffect(() => {
		let alive = true;
		const load = (value: boolean) => {
			if (!alive) return;
			enabled.value = value;
			ready.value = true;
		};
		DiscordEnabled().then(load, () => load(false));
		return () => {
			alive = false;
		};
	}, []);

	async function toggle() {
		if (!ready.value) return;
		const next = !enabled.value;
		enabled.value = next;
		try {
			await SetDiscordEnabled(next);
			if (next) media.refresh();
		} catch {
			enabled.value = !next;
		}
	}

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center justify-between gap-2">
				<div class="flex items-center gap-2">
					<Headphones size={18} class="text-zinc-400" />
					<h2 class="text-sm font-semibold text-zinc-200">
						{t("settings.discord.title")}
					</h2>
				</div>
				<button
					type="button"
					role="switch"
					aria-checked={enabled.value}
					aria-label={t("settings.discord.label")}
					disabled={!ready.value}
					onClick={() => void toggle()}
					class={cn(
						"relative h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors disabled:opacity-50",
						enabled.value ? "bg-indigo-500" : "bg-zinc-700",
					)}
				>
					<span
						class={cn(
							"absolute left-0.5 top-0.5 size-5 rounded-full bg-white transition-transform",
							enabled.value ? "translate-x-5" : "translate-x-0",
						)}
					/>
				</button>
			</div>
			<p class="text-xs text-zinc-500">{t("settings.discord.description")}</p>
		</section>
	);
}

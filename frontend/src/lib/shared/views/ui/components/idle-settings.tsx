import { IdleReload, SetIdleReload } from "@bindings/app";
import { useSignal } from "@preact/signals";
import { Moon } from "lucide-preact";
import { useEffect } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { Switch } from "@/lib/shared/views/ui/components/switch";

export function IdleSettings() {
	const enabled = useSignal(false);
	const ready = useSignal(false);

	useEffect(() => {
		let alive = true;
		const load = (value: boolean) => {
			if (!alive) return;
			enabled.value = value;
			ready.value = true;
		};
		IdleReload().then(load, () => load(false));
		return () => {
			alive = false;
		};
	}, []);

	async function toggle() {
		if (!ready.value) return;
		const next = !enabled.value;
		enabled.value = next;
		try {
			await SetIdleReload(next);
		} catch {
			enabled.value = !next;
		}
	}

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center justify-between gap-2">
				<div class="flex items-center gap-2">
					<Moon size={18} class="text-zinc-400" />
					<h2 class="text-sm font-semibold text-zinc-200">
						{t("settings.idle.title")}
					</h2>
				</div>
				<Switch
					checked={enabled.value}
					label={t("settings.idle.label")}
					disabled={!ready.value}
					onToggle={() => void toggle()}
				/>
			</div>
			<p class="text-xs text-zinc-500">{t("settings.idle.description")}</p>
		</section>
	);
}

export default IdleSettings;

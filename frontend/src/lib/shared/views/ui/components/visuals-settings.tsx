import { Palette } from "lucide-preact";
import { backdrop, setBackdrop } from "@/lib/music/views/stores/visuals";
import { t } from "@/lib/shared/i18n";
import { Switch } from "@/lib/shared/views/ui/components/switch";

export function VisualsSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Palette size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("settings.visuals.title")}
				</h2>
			</div>

			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<p class="text-sm text-zinc-200">
						{t("settings.visuals.background.label")}
					</p>
					<p class="text-xs text-zinc-500">
						{t("settings.visuals.background.description")}
					</p>
				</div>
				<Switch
					checked={backdrop.value}
					label={t("settings.visuals.background.label")}
					onToggle={() => setBackdrop(!backdrop.value)}
				/>
			</div>
		</section>
	);
}

export default VisualsSettings;

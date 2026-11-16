import { IdleReload, SetIdleReload } from "@bindings/app";
import { Moon } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { ToggleSetting } from "@/lib/shared/views/ui/components/toggle-setting";

export function IdleSettings() {
	return (
		<ToggleSetting
			icon={<Moon size={18} class="text-zinc-400" />}
			title={t("settings.idle.title")}
			description={t("settings.idle.description")}
			label={t("settings.idle.label")}
			load={IdleReload}
			save={SetIdleReload}
		/>
	);
}

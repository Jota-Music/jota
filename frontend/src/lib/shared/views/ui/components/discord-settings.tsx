import { DiscordEnabled, SetDiscordEnabled } from "@bindings/app";
import { Headphones } from "lucide-preact";
import * as media from "@/lib/music/views/stores/media-session";
import { t } from "@/lib/shared/i18n";
import { ToggleSetting } from "@/lib/shared/views/ui/components/toggle-setting";

export function DiscordSettings() {
	return (
		<ToggleSetting
			icon={<Headphones size={18} class="text-zinc-400" />}
			title={t("settings.discord.title")}
			description={t("settings.discord.description")}
			label={t("settings.discord.label")}
			load={DiscordEnabled}
			save={SetDiscordEnabled}
			changed={(enabled) => {
				if (enabled) media.refresh();
			}}
		/>
	);
}

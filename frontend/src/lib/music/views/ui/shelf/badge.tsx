import { Library } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";
import type { Source } from "./types";

export function SourceBadge({ source }: { source?: Source }) {
	if (!source) return null;
	const title =
		source === "spotify"
			? t("music.shelf.source.spotify")
			: source === "youtube"
				? t("music.shelf.source.youtube")
				: t("music.custom.tab");
	return (
		<span
			class="flex size-5 items-center justify-center text-zinc-400"
			title={title}
		>
			{source === "spotify" ? (
				<SpotifyIcon size={14} />
			) : source === "youtube" ? (
				<YoutubeIcon class="size-4" />
			) : (
				<Library size={14} />
			)}
		</span>
	);
}

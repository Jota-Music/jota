import { Edit2 } from "lucide-preact";
import type { Song } from "@/lib/music/model";
import { openYoutubeEditor } from "@/lib/music/views/stores/youtube-editor";
import { t } from "@/lib/shared/i18n";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export default function SaveYoutubeId({ song }: { song: Song }) {
	return (
		<button
			type="button"
			class="group mt-2 flex h-8 items-center gap-2"
			onClick={() => openYoutubeEditor(song)}
		>
			<YoutubeIcon class="size-4 text-white/20" />
			<span class="flex flex-1 items-center gap-2 truncate text-xs text-white opacity-50">
				<span class="rounded bg-white/10 px-1.5 py-0.5 font-mono">
					{song.youtubeId || "..."}
				</span>
			</span>
			{song.youtubeId && (
				<span
					class="cursor-pointer rounded p-1 opacity-0 transition hover:brightness-200 group-hover:opacity-100"
					title={t("music.youtubeId.edit")}
				>
					<Edit2 class="size-4" />
				</span>
			)}
		</button>
	);
}

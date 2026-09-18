import { SquarePlus } from "lucide-preact";
import type { Song } from "@/lib/music/model";
import { openPicker } from "@/lib/music/views/stores/selection";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

export default function AddToPlaylist({
	song,
	class: className,
}: {
	song: Song;
	class?: string;
}) {
	return (
		<button
			type="button"
			title={t("music.custom.addTo")}
			aria-label={t("music.custom.addTo")}
			onClick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				openPicker([song]);
			}}
			class={cn(
				"shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-(--dominant-color)",
				className,
			)}
		>
			<SquarePlus size={18} strokeWidth={2} />
		</button>
	);
}

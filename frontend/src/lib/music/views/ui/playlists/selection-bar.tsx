import { SquarePlus, X } from "lucide-preact";
import {
	clearSelection,
	openPicker,
	selectedSongs,
	selectionActive,
} from "@/lib/music/views/stores/selection";
import { t } from "@/lib/shared/i18n";

export function SelectionBar() {
	if (!selectionActive.value) return null;

	const songs = selectedSongs.value;

	return (
		<div class="pointer-events-none absolute inset-x-0 bottom-4 z-40 flex justify-center px-4">
			<div class="pointer-events-auto flex items-center gap-1 rounded-full border border-zinc-700 bg-zinc-900/95 py-1 pl-3 pr-1 shadow-xl backdrop-blur">
				<span class="mr-1 text-xs tabular-nums text-zinc-300">
					{t("music.custom.selected", { count: songs.length })}
				</span>
				<button
					type="button"
					onClick={() => openPicker(songs)}
					class="flex cursor-pointer items-center gap-1.5 rounded-full bg-(--dominant-color) px-3 py-1.5 text-xs font-medium text-(--binary-color) transition-opacity hover:opacity-80"
				>
					<SquarePlus size={14} />
					{t("music.custom.addTo")}
				</button>
				<button
					type="button"
					title={t("music.custom.clear")}
					aria-label={t("music.custom.clear")}
					onClick={clearSelection}
					class="flex size-8 cursor-pointer items-center justify-center rounded-full text-zinc-400 transition hover:bg-zinc-800 hover:text-white"
				>
					<X size={16} />
				</button>
			</div>
		</div>
	);
}

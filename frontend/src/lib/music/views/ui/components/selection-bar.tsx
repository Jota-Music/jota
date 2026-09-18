import { Check, X } from "lucide-preact";
import type { Song } from "@/lib/music/model";
import {
	clearSelection,
	selectAll,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import TrackActions from "@/lib/music/views/ui/components/track-actions";
import { t } from "@/lib/shared/i18n";

export default function SelectionBar({
	removable,
	onRemove,
}: {
	removable?: boolean;
	onRemove?: (songs: Song[]) => void;
}) {
	const selected = selectedSongs.value;
	if (selected.length <= 1) return null;

	return (
		<div class="flex h-10 shrink-0 items-center gap-2 rounded-md border border-zinc-800 bg-zinc-950 px-3">
			<span class="flex size-5 shrink-0 items-center justify-center rounded-full border-2 border-(--dominant-color) bg-(--dominant-color) text-(--binary-color)">
				<Check size={13} strokeWidth={3} />
			</span>
			<span class="text-sm text-zinc-300">
				{t("music.custom.selected", { count: selected.length })}
			</span>
			<div class="ml-auto flex items-center gap-1">
				<TrackActions
					songs={selected}
					onSelectAll={selectAll}
					removable={removable}
					onRemove={onRemove}
				/>
				<button
					type="button"
					title={t("music.custom.clear")}
					aria-label={t("music.custom.clear")}
					onClick={clearSelection}
					class="flex size-8 cursor-pointer items-center justify-center rounded-md text-zinc-400 transition hover:bg-zinc-800/80 hover:text-white"
				>
					<X size={16} />
				</button>
			</div>
		</div>
	);
}

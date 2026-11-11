import { Check, X } from "lucide-preact";
import {
	clearSelection,
	remover,
	selectAll,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import TrackActions from "@/lib/music/views/ui/track/track-actions";
import { t } from "@/lib/shared/i18n";

// The banner under the header, next to the update and clipboard ones: it
// belongs to the shell, so every list that selects songs gets it.
export default function SelectionBar() {
	const selected = selectedSongs.value;
	if (selected.length <= 1) return null;

	return (
		<div class="flex items-center gap-3 border-b border-zinc-800 bg-zinc-900/60 px-4 py-2 text-sm">
			<span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-(--dominant-color) text-(--binary-color)">
				<Check size={14} strokeWidth={3} />
			</span>
			<p class="min-w-0 flex-1 truncate text-zinc-400">
				{t("music.custom.selected", { count: selected.length })}
			</p>
			<TrackActions
				songs={selected}
				onSelectAll={selectAll}
				removable={!!remover.value}
				onRemove={remover.value ?? undefined}
			/>
			<button
				type="button"
				title={t("music.custom.clear")}
				aria-label={t("music.custom.clear")}
				onClick={clearSelection}
				class="shrink-0 cursor-pointer text-zinc-500 transition-colors hover:text-zinc-200"
			>
				<X size={14} />
			</button>
		</div>
	);
}

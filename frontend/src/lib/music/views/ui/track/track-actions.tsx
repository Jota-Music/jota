import {
	EllipsisVertical,
	Link2,
	ListChecks,
	ListPlus,
	Pause,
	Play,
	Radio,
	Square,
	SquareCheck,
	SquarePlus,
	Trash2,
	Volume2,
	VolumeX,
} from "lucide-preact";
import { useLocation } from "wouter-preact";
import { playlistSource } from "@/lib/music/app/playlist-source";
import type { Song } from "@/lib/music/model";
import {
	currentSong,
	isPlaying,
	muted,
	toggleMute,
} from "@/lib/music/views/stores/audio";
import {
	enqueueMany,
	playList,
	toggleSong,
} from "@/lib/music/views/stores/player";
import {
	clearSelection,
	openPicker,
	selectedIds,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import { openEditor } from "@/lib/music/views/stores/youtube-editor";
import { shareActions } from "@/lib/music/views/ui/share";
import { t } from "@/lib/shared/i18n";
import {
	type Action,
	openContextMenu,
} from "@/lib/shared/views/ui/components/context-menu";

type Props = {
	songs: Song[];
	// playlistActions are about the containing playlist, not its tracks, and
	// lead the same menu.
	playlistActions?: Action[];
	// share is always last: it acts on the page, not on the selection.
	share?: Action[];
	selection?: boolean;
	removable?: boolean;
	onPlay?: () => void;
	onRemove?: (songs: Song[]) => void;
	onToggleSelect?: (song: Song) => void;
	onSelectAll?: (songs: Song[]) => void;
	onDone?: () => void;
	navigate?: (to: string) => void;
	// reserve keeps the button in place while the list is still loading, so the
	// header it sits in does not shift when the tracks land.
	reserve?: boolean;
	// Clearance below the viewport bottom for a fixed element anchored there
	// (the compact player bar) when this button lives inside it.
	bottom?: number;
	// volume adds the mute switch the compact bar used to carry as its own
	// control, for the platform where the bar has no room for it.
	volume?: boolean;
};

export function buildActions({
	songs,
	playlistActions,
	share,
	selection,
	removable,
	onPlay,
	onRemove,
	onToggleSelect,
	onSelectAll,
	onDone,
	navigate,
	volume,
}: Props): { actions: Action[]; count: number } {
	const batch =
		selection && selectedSongs.value.length > 1 && songs.length === 1;
	const list = (batch ? selectedSongs.value : songs).filter(
		(song) => !song.broken,
	);
	if (list.length === 0)
		return { actions: [...(playlistActions ?? [])], count: 0 };

	const single = list.length === 1 ? list[0] : null;
	const done = () => void onDone?.();

	// The track that is already playing pauses instead of restarting.
	const playing =
		!!single && currentSong.value?.id === single.id && isPlaying.value;

	// Playlist actions come first: they act on the whole list, while the rest
	// act on the tracks inside it.
	const actions: Action[] = [...(playlistActions ?? [])];
	actions.push(
		{
			icon: playing ? Pause : Play,
			label: playing ? t("music.pause") : t("music.play"),
			run: () => {
				// Activating the loaded track restarts it, so the pause action has
				// to toggle on its own.
				if (playing) void toggleSong();
				else if (onPlay) onPlay();
				else void playList(list);
				done();
			},
		},
		{
			icon: ListPlus,
			label: t("music.track.addQueue"),
			run: () => {
				enqueueMany(list);
				done();
			},
		},
		{
			icon: SquarePlus,
			label: t("music.custom.addTo"),
			run: () => {
				openPicker(list);
				done();
			},
		},
	);
	if ((onSelectAll || batch) && list.length > 1) {
		const allSelected = list.every((song) => selectedIds.value.has(song.id));
		actions.push({
			icon: allSelected ? Square : ListChecks,
			label: allSelected
				? t("music.custom.deselectAll")
				: t("music.custom.selectAll"),
			run: () => {
				if (allSelected) clearSelection();
				else onSelectAll?.(list);
				done();
			},
		});
	}
	if (onToggleSelect && single) {
		const isSelected = selectedIds.value.has(single.id);
		actions.push({
			icon: isSelected ? Square : SquareCheck,
			label: isSelected ? t("music.custom.deselect") : t("music.custom.select"),
			run: () => {
				onToggleSelect(single);
				done();
			},
		});
	}
	if (single) {
		if (playlistSource(single.id) === "spotify" && navigate) {
			actions.push({
				icon: Radio,
				label: t("music.radio.go"),
				run: () => {
					navigate(`/radio/${single.id}`);
					done();
				},
			});
		}
		actions.push({
			icon: Link2,
			label: t("music.youtubeId.add"),
			run: () => {
				openEditor(single);
				done();
			},
		});
	}
	if (removable && onRemove) {
		actions.push({
			icon: Trash2,
			label: t("music.custom.removeSong"),
			danger: true,
			run: () => {
				onRemove(list);
				done();
			},
		});
	}
	if (volume) {
		actions.push({
			icon: muted.value ? VolumeX : Volume2,
			label: muted.value ? t("music.volume.unmute") : t("music.volume.mute"),
			run: () => {
				toggleMute();
				done();
			},
		});
	}
	if (single) actions.push(...shareActions(single));
	if (share) actions.push(...share);

	return { actions, count: batch && list.length > 1 ? list.length : 0 };
}

// Context menu for a single song without its own trigger button: right-click
// on the player. buildActions stays the single source of menu contents.
export function useTrackMenu(
	song: Song | undefined,
	onPlay?: () => void,
	bottom = 0,
	volume = false,
) {
	const [, navigate] = useLocation();
	return (e: MouseEvent) => {
		if (!song) return;
		const { actions } = buildActions({
			songs: [song],
			onPlay,
			navigate,
			volume,
		});
		if (actions.length > 0) openContextMenu(actions, e, undefined, bottom);
	};
}

export default function TrackActions({
	songs,
	playlistActions,
	share,
	selection,
	removable,
	onPlay,
	onRemove,
	onToggleSelect,
	onSelectAll,
	onDone,
	reserve = false,
	bottom = 0,
	volume = false,
}: Props) {
	const [, navigate] = useLocation();

	const { actions, count } = buildActions({
		songs,
		playlistActions,
		share,
		selection,
		removable,
		onPlay,
		onRemove,
		onToggleSelect,
		onSelectAll,
		onDone,
		navigate,
		volume,
	});

	const empty = actions.length === 0;
	if (empty && !reserve) return null;

	const toggle = (e: MouseEvent) => {
		if (empty) return;
		e.stopPropagation();
		openContextMenu(
			actions,
			e,
			count > 0 ? t("music.custom.selected", { count }) : undefined,
			bottom,
		);
	};

	return (
		<button
			type="button"
			title={t("music.track.actions")}
			aria-label={t("music.track.actions")}
			aria-haspopup="menu"
			disabled={empty}
			onPointerDown={(e) => {
				e.preventDefault();
				e.stopPropagation();
			}}
			onClick={toggle}
			class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:text-white disabled:cursor-default disabled:opacity-40"
		>
			<EllipsisVertical size={18} />
		</button>
	);
}

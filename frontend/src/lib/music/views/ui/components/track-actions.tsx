import {
	EllipsisVertical,
	ListChecks,
	ListPlus,
	type LucideIcon,
	Square,
	SquareCheck,
	SquarePlus,
	Trash2,
} from "lucide-preact";
import { createPortal } from "preact/compat";
import { useLayoutEffect, useRef, useState } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { enqueueMany } from "@/lib/music/views/stores/player";
import {
	clearSelection,
	openPicker,
	selectedSongs,
} from "@/lib/music/views/stores/selection";
import { openYoutubeEditor } from "@/lib/music/views/stores/youtube-editor";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

const YoutubeMenuIcon: LucideIcon = ({ class: cls, size }) => (
	<YoutubeIcon
		class={cls as string | undefined}
		width={typeof size === "number" ? size : undefined}
		height={typeof size === "number" ? size : undefined}
	/>
);

type Action = {
	icon: LucideIcon | typeof YoutubeMenuIcon;
	label: string;
	danger?: boolean;
	run: () => void;
};

type Props = {
	songs: Song[];
	removable?: boolean;
	onRemove?: (songs: Song[]) => void;
	onToggleSelect?: (song: Song) => void;
	onSelectAll?: (songs: Song[]) => void;
	onDone?: () => void;
};

const GAP = 6;

export default function TrackActions({
	songs,
	removable,
	onRemove,
	onToggleSelect,
	onSelectAll,
	onDone,
}: Props) {
	const [open, setOpen] = useState(false);
	const [anchor, setAnchor] = useState<DOMRect | null>(null);
	const button = useRef<HTMLButtonElement>(null);
	const menu = useRef<HTMLDivElement>(null);

	const list = songs.filter((song) => !song.broken);
	if (list.length === 0) return null;

	const single = list.length === 1 ? list[0] : null;
	const done = () => void onDone?.();

	const close = () => {
		setOpen(false);
		setAnchor(null);
	};

	const actions: Action[] = [
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
	];
	if (onSelectAll && list.length > 1) {
		const allSelected = list.every((song) =>
			selectedSongs.value.some((s) => s.id === song.id),
		);
		actions.push({
			icon: allSelected ? Square : ListChecks,
			label: allSelected
				? t("music.custom.deselectAll")
				: t("music.custom.selectAll"),
			run: () => {
				if (allSelected) clearSelection();
				else onSelectAll(list);
				done();
			},
		});
	}
	if (onToggleSelect && single) {
		const isSelected = selectedSongs.value.some((s) => s.id === single.id);
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
		actions.push({
			icon: YoutubeMenuIcon,
			label: single.youtubeId
				? t("music.youtubeId.edit")
				: t("music.youtubeId.add"),
			run: () => {
				openYoutubeEditor(single);
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

	useLayoutEffect(() => {
		if (!open) return;
		const el = menu.current;
		if (!el || !anchor) return;

		const width = el.offsetWidth;
		const below = anchor.bottom + GAP;
		const fits = below + el.offsetHeight <= window.innerHeight - GAP;
		el.style.left = `${anchor.right - width}px`;
		el.style.top = `${fits ? below : Math.max(GAP, anchor.top - el.offsetHeight - GAP)}px`;
		el.querySelector("button")?.focus();

		const onDown = (e: PointerEvent) => {
			const target = e.target as Node;
			if (!el.contains(target) && !button.current?.contains(target)) close();
		};
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") {
				close();
				button.current?.focus();
			}
		};
		const onScroll = () => close();
		const onResize = () => close();
		document.addEventListener("pointerdown", onDown, true);
		document.addEventListener("keydown", onKey);
		window.addEventListener("scroll", onScroll, true);
		window.addEventListener("resize", onResize);
		return () => {
			document.removeEventListener("pointerdown", onDown, true);
			document.removeEventListener("keydown", onKey);
			window.removeEventListener("scroll", onScroll, true);
			window.removeEventListener("resize", onResize);
		};
	}, [open, anchor]);

	const toggle = (e: MouseEvent) => {
		e.preventDefault();
		e.stopPropagation();
		const rect = button.current?.getBoundingClientRect();
		if (!rect) return;
		if (open) close();
		else {
			setAnchor(rect);
			setOpen(true);
		}
	};

	return (
		<>
			<button
				ref={button}
				type="button"
				title={t("music.track.actions")}
				aria-label={t("music.track.actions")}
				aria-haspopup="menu"
				aria-expanded={open}
				onPointerDown={(e) => {
					e.preventDefault();
					e.stopPropagation();
				}}
				onClick={toggle}
				class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-white"
			>
				<EllipsisVertical size={18} />
			</button>

			{open &&
				anchor &&
				createPortal(
					<div
						ref={menu}
						role="menu"
						aria-label={t("music.track.actions")}
						class="fixed z-50 w-56 rounded-lg border border-zinc-800 bg-zinc-950 py-1 shadow-xl"
						style={{ top: 0, left: 0 }}
					>
						{actions.map((action) => (
							<button
								key={action.label}
								type="button"
								role="menuitem"
								onClick={() => {
									close();
									action.run();
								}}
								class={cn(
									"flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-sm transition-colors",
									action.danger
										? "text-red-400 hover:bg-red-950/40"
										: "text-zinc-300 hover:bg-zinc-800/80 hover:text-white",
								)}
							>
								<action.icon
									size={16}
									class={action.danger ? "text-red-400" : "text-zinc-500"}
								/>
								{action.label}
							</button>
						))}
					</div>,
					document.body,
				)}
		</>
	);
}

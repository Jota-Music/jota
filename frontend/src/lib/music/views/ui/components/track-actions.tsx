import {
	EllipsisVertical,
	ListPlus,
	type LucideIcon,
	SquarePlus,
	Trash2,
} from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect, useRef, useState } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { enqueue } from "@/lib/music/views/stores/player";
import { openPicker } from "@/lib/music/views/stores/selection";
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
	song: Song;
	removable?: boolean;
	onRemove?: (song: Song) => void;
};

const GAP = 6;

export default function TrackActions({ song, removable, onRemove }: Props) {
	const [open, setOpen] = useState(false);
	const [anchor, setAnchor] = useState<DOMRect | null>(null);
	const button = useRef<HTMLButtonElement>(null);
	const menu = useRef<HTMLDivElement>(null);

	if (song.broken) return null;

	const close = () => {
		setOpen(false);
		setAnchor(null);
	};

	const actions: Action[] = [
		{
			icon: ListPlus,
			label: t("music.track.addQueue"),
			run: () => enqueue(song),
		},
		{
			icon: SquarePlus,
			label: t("music.custom.addTo"),
			run: () => openPicker([song]),
		},
		{
			icon: YoutubeMenuIcon,
			label: song.youtubeId
				? t("music.youtubeId.edit")
				: t("music.youtubeId.add"),
			run: () => openYoutubeEditor(song),
		},
	];
	if (removable && onRemove) {
		actions.push({
			icon: Trash2,
			label: t("music.custom.removeSong"),
			danger: true,
			run: () => onRemove(song),
		});
	}

	useEffect(() => {
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
			if (!el.contains(target) && target !== button.current) close();
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

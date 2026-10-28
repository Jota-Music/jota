import {
	Disc3,
	Link2,
	ListMusic,
	Loader2,
	Play,
	Tv,
	UserRound,
	X,
} from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import type { ClipboardLink } from "@/lib/clipboard/app/link";
import { dismiss, pending } from "@/lib/clipboard/views/stores/pending";
import { getSong } from "@/lib/music/app/get-song";
import {
	currentSong,
	isLoading,
	isPlaying,
	pendingStart,
} from "@/lib/music/views/stores/audio";
import { playList } from "@/lib/music/views/stores/player";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";

function label(link: ClipboardLink): string {
	switch (link.kind) {
		case "song":
			return link.to.startsWith("youtube:")
				? t("clipboard.track")
				: t("clipboard.track");
		case "user":
			return t("clipboard.channel");
		default:
			return t(`clipboard.${link.kind}`);
	}
}

function routeKey(path: string): string {
	return path.replace(/\/+$/, "") || "/";
}

function isCurrentSong(link: ClipboardLink | undefined): boolean {
	if (!link || link.kind !== "song") return false;
	const song = currentSong.value;
	if (!song) return false;

	if (link.to === song.id || link.to === song.youtubeId) return true;
	return (
		link.to.startsWith("youtube:") &&
		link.to.slice("youtube:".length) === song.youtubeId
	);
}

export function ClipboardBanner() {
	const [location, setLocation] = useLocation();
	const [busy, setBusy] = useState(false);

	const current = pending.value;
	const link = current?.link;
	const alreadyAtDestination =
		link !== undefined &&
		link.kind !== "song" &&
		routeKey(location) === routeKey(link.to);
	const alreadyPlayingSong =
		isCurrentSong(link) &&
		(isPlaying.value || isLoading.value || pendingStart.value);
	const redundant = alreadyAtDestination || alreadyPlayingSong;

	useEffect(() => {
		if (redundant && link && pending.value?.link.to === link.to) {
			pending.value = null;
		}
	}, [redundant, link?.to]);

	if (!current || current.dismissed || redundant || !link) return null;

	const playable = link.kind === "song";
	const Icon =
		link.kind === "song"
			? Play
			: link.kind === "album"
				? Disc3
				: link.kind === "playlist"
					? ListMusic
					: link.kind === "channel"
						? Tv
						: UserRound;

	const open = async () => {
		if (!playable) {
			pending.value = null;
			setLocation(link.to);
			return;
		}
		setBusy(true);
		try {
			playList([await getSong(link.to)]);
			pending.value = null;
		} catch (error) {
			addError(error, "clipboard");
		} finally {
			setBusy(false);
		}
	};

	return (
		<div class="relative flex items-center gap-3 border-b border-zinc-800 bg-zinc-900/60 px-4 py-2 text-sm">
			{/* The whole bar acts, not only the round button, so a tap anywhere but
			    the dismiss reaches the same action. The icon stays a span: the
			    button keeps its look without nesting a button inside a button. */}
			<button
				type="button"
				onClick={() => void open()}
				disabled={busy}
				title={label(link)}
				aria-label={label(link)}
				class="-mx-4 -my-2 flex min-w-0 flex-1 cursor-pointer items-center gap-3 px-4 py-2 text-left disabled:cursor-default"
			>
				<Link2 size={16} class="shrink-0 text-(--dominant-color)" />

				<p class="min-w-0 flex-1 truncate text-zinc-400">{link.text}</p>

				{busy ? (
					<span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-zinc-800 text-zinc-300">
						<Loader2 size={14} class="animate-spin" />
					</span>
				) : (
					<span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-(--dominant-color) text-(--binary-color) transition-opacity hover:opacity-85">
						<Icon size={14} class={playable ? "fill-current" : undefined} />
					</span>
				)}
			</button>

			<button
				type="button"
				onClick={() => dismiss()}
				aria-label={t("common.dismiss")}
				class="shrink-0 cursor-pointer text-zinc-500 transition-colors hover:text-zinc-200"
			>
				<X size={14} />
			</button>
		</div>
	);
}

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
import { useState } from "preact/hooks";
import { useLocation } from "wouter-preact";
import type { ClipboardLink } from "@/lib/clipboard/app/link";
import { dismiss, pending } from "@/lib/clipboard/views/stores/pending";
import { getSong } from "@/lib/music/app/get-song";
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

export function ClipboardBanner() {
	const [, setLocation] = useLocation();
	const [busy, setBusy] = useState(false);

	const current = pending.value;
	if (!current || current.dismissed) return null;

	const { link } = current;
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
			<Link2 size={16} class="shrink-0 text-(--dominant-color)" />

			<p class="min-w-0 flex-1 truncate text-zinc-400">{link.text}</p>

			{busy ? (
				<span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-zinc-800 text-zinc-300">
					<Loader2 size={14} class="animate-spin" />
				</span>
			) : (
				<button
					type="button"
					onClick={() => void open()}
					title={label(link)}
					aria-label={label(link)}
					class="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-full bg-(--dominant-color) text-(--binary-color) transition-opacity hover:opacity-85"
				>
					<Icon size={14} class={playable ? "fill-current" : undefined} />
				</button>
			)}

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

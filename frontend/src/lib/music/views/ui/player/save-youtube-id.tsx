import { useSignal } from "@preact/signals";
import { Check, Edit2, X } from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { updateYoutubeId } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";
import {
	currentSong,
	getPlaybackSeconds,
	isPlaying,
	play,
	prepareSong,
} from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export default function SaveYoutubeId({
	song,
	compact = false,
}: {
	song: Song;
	compact?: boolean;
}) {
	const [open, setOpen] = useState(false);
	const youtubeId = useSignal(song.youtubeId ?? "");
	const $input = useRef<HTMLInputElement>(null);

	const start = () => {
		youtubeId.value = song.youtubeId ?? "";
		setOpen(true);
	};

	useEffect(() => {
		if (open) {
			$input.current?.focus({ preventScroll: true });
			$input.current?.select();
		}
	}, [open]);

	async function handlSubmit(e: Event) {
		e.preventDefault();
		const formData = new FormData(e.target as HTMLFormElement);
		const youtube = formData.get("youtube") as string;

		if (song.youtubeId === youtube) {
			setOpen(false);
			return;
		}

		await updateYoutubeId(song.id, youtube);
		song.youtubeId = youtube;
		youtubeId.value = youtube;
		AudioCache.remove(song.id);

		if (currentSong.value?.id === song.id) {
			const at = getPlaybackSeconds();
			if (isPlaying.value) {
				await play(song, at);
			} else {
				await prepareSong(song, at);
			}
		}

		setOpen(false);
	}

	return (
		<>
			{compact ? (
				<button
					type="button"
					title={
						song.youtubeId
							? `Edit YouTube ID (${song.youtubeId})`
							: "Add YouTube ID"
					}
					class="cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-amber-300"
					onClick={(e) => {
						e.stopPropagation();
						start();
					}}
				>
					<YoutubeIcon class="size-4" />
				</button>
			) : (
				<button
					type="button"
					class="group mt-2 flex h-8 items-center gap-2"
					onClick={start}
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
							title="Edit YouTube ID"
						>
							<Edit2 class="size-4" />
						</span>
					)}
				</button>
			)}

			<Sheet
				open={open}
				close={() => setOpen(false)}
				labelledBy="youtube-id-title"
				closeLabel="Close YouTube ID editor"
			>
				<form class="flex flex-col gap-4 p-5" onSubmit={handlSubmit}>
					<div class="flex flex-col gap-1">
						<h2 id="youtube-id-title" class="text-sm font-semibold text-white">
							YouTube ID
						</h2>
						<p class="truncate text-xs text-zinc-500">{song.name}</p>
					</div>

					<input
						ref={$input}
						type="text"
						value={youtubeId.value}
						name="youtube"
						placeholder="YouTube ID"
						class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm text-white outline-none focus:border-zinc-600"
						onInput={(e) => {
							youtubeId.value = (e.target as HTMLInputElement).value;
						}}
						onKeyDown={(e) => {
							if (e.key === "Escape") {
								setOpen(false);
							}
						}}
					/>

					<div class="flex justify-end gap-2">
						<button
							type="button"
							title="Cancel"
							aria-label="Cancel"
							class="cursor-pointer rounded-md p-2 text-zinc-400 transition hover:bg-zinc-800 hover:text-white"
							onClick={() => setOpen(false)}
						>
							<X class="size-4" />
						</button>
						<button
							type="submit"
							title="Save"
							aria-label="Save"
							class="cursor-pointer rounded-md p-2 text-white transition hover:bg-zinc-700"
						>
							<Check class="size-4" />
						</button>
					</div>
				</form>
			</Sheet>
		</>
	);
}

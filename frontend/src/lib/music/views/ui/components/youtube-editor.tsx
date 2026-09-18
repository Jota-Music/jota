import { useSignal } from "@preact/signals";
import { Check } from "lucide-preact";
import { useEffect, useRef } from "preact/hooks";
import { getYouTubeId, updateYoutubeId } from "@/lib/music/app/get-audio";
import { currentSong, setYoutube } from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import { reloadCurrent } from "@/lib/music/views/stores/player";
import {
	closeYoutubeEditor,
	editingSong,
} from "@/lib/music/views/stores/youtube-editor";
import { t } from "@/lib/shared/i18n";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export default function YoutubeEditor() {
	const song = editingSong.value;
	const open = song != null;
	const youtubeId = useSignal("");
	const $input = useRef<HTMLInputElement>(null);

	useEffect(() => {
		if (!open || !song) return;
		youtubeId.value = song.youtubeId ?? "";
		$input.current?.focus({ preventScroll: true });
		$input.current?.select();

		let active = true;
		if (!song.youtubeId) {
			getYouTubeId(song.id)
				.then((id) => {
					if (active) youtubeId.value = id;
				})
				.catch(() => {});
		}
		return () => {
			active = false;
		};
	}, [open, song]);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!song) return;
		const formData = new FormData(e.target as HTMLFormElement);
		const youtube = formData.get("youtube") as string;

		if (song.youtubeId === youtube) {
			closeYoutubeEditor();
			return;
		}

		await updateYoutubeId(song.id, youtube);
		setYoutube(song, youtube);
		youtubeId.value = youtube;
		AudioCache.remove(song.id);

		if (currentSong.value?.id === song.id) {
			await reloadCurrent();
		}

		closeYoutubeEditor();
	}

	return (
		<Modal
			open={open}
			close={closeYoutubeEditor}
			labelledBy="youtube-id-title"
			closeLabel={t("music.youtubeId.close")}
		>
			{song && (
				<form class="flex flex-col gap-4 p-5" onSubmit={handleSubmit}>
					<div class="flex flex-col gap-1">
						<h2 id="youtube-id-title" class="text-sm font-semibold text-white">
							{t("music.youtubeId.title")}
						</h2>
						<p class="truncate text-xs text-zinc-500">{song.name}</p>
						<p class="text-balance text-xs text-zinc-500">
							{t("music.youtubeId.hint")}
						</p>
					</div>

					<div class="flex items-center gap-2">
						<YoutubeIcon class="size-4 shrink-0 text-white/30" />
						<input
							ref={$input}
							type="text"
							value={youtubeId.value}
							name="youtube"
							placeholder={t("music.youtubeId.placeholder")}
							class="w-full rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm text-white outline-none focus:border-zinc-600"
							onInput={(e) => {
								youtubeId.value = (e.target as HTMLInputElement).value;
							}}
							onKeyDown={(e) => {
								if (e.key === "Escape") {
									closeYoutubeEditor();
								}
							}}
						/>
						<button
							type="submit"
							title={t("music.youtubeId.save")}
							aria-label={t("music.youtubeId.save")}
							class="cursor-pointer self-center rounded-md p-2 text-white transition hover:bg-zinc-700"
						>
							<Check class="size-4" />
						</button>
					</div>
				</form>
			)}
		</Modal>
	);
}

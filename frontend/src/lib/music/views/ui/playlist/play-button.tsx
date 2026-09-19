import { useSignal } from "@preact/signals";
import { useQueryClient } from "@tanstack/preact-query";
import { CirclePlay, Loader } from "lucide-preact";
import { getFullPlaylist } from "@/lib/music/app/get-playlist";
import { playList } from "@/lib/music/views/stores/player";
import { t } from "@/lib/shared/i18n";

const buttonClass =
	"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300 transition hover:bg-black/90 hover:text-white disabled:opacity-50";

export function PlaylistPlayButton({ id }: { id: string }) {
	const queryClient = useQueryClient();
	const busy = useSignal(false);

	async function play() {
		if (busy.value) return;
		busy.value = true;
		try {
			const playlist = await queryClient.fetchQuery({
				queryKey: ["playlist", id],
				queryFn: () => getFullPlaylist(id),
			});
			const songs = playlist.songs.filter((song) => !song.broken);
			if (songs.length > 0) playList(songs);
		} finally {
			busy.value = false;
		}
	}

	return (
		<button
			type="button"
			title={t("music.playlist.play")}
			aria-label={t("music.playlist.play")}
			disabled={busy.value}
			onClick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				void play();
			}}
			class={buttonClass}
		>
			{busy.value ? (
				<Loader size={16} class="animate-spin" />
			) : (
				<CirclePlay size={16} />
			)}
		</button>
	);
}

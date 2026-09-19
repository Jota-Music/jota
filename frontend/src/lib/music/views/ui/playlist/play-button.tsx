import { CirclePlay, Loader, Pause } from "lucide-preact";
import { loadingPlaylist, playPlaylist } from "@/lib/music/views/play";
import { isLoading, isPlaying } from "@/lib/music/views/stores/audio";
import { queueSource } from "@/lib/music/views/stores/queue";
import { t } from "@/lib/shared/i18n";

const buttonClass =
	"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 text-zinc-300 transition hover:bg-black/90 hover:text-white disabled:opacity-50";

export function PlaylistPlayButton({ id }: { id: string }) {
	const loading =
		loadingPlaylist.value === id ||
		(queueSource.value === id && isLoading.value);
	const playing = queueSource.value === id && isPlaying.value;
	const label = playing ? t("music.pause") : t("music.play");

	return (
		<button
			type="button"
			title={label}
			aria-label={label}
			disabled={loading}
			onClick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				void playPlaylist(id);
			}}
			class={buttonClass}
		>
			{loading ? (
				<Loader size={16} class="animate-spin" />
			) : playing ? (
				<Pause size={16} class="fill-current" />
			) : (
				<CirclePlay size={16} />
			)}
		</button>
	);
}

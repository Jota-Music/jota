import { Clipboard } from "@wailsio/runtime";
import { Link2, type LucideIcon } from "lucide-preact";
import { remember } from "@/lib/clipboard/views/stores/pending";
import { resolveYoutubeId } from "@/lib/music/app/get-audio";
import { canShare, spotifyLink } from "@/lib/music/app/share";
import type { Song } from "@/lib/music/model";
import { setYoutube } from "@/lib/music/views/stores/audio";
import { t } from "@/lib/shared/i18n";
import { addError, notify } from "@/lib/shared/views/stores/errors";
import type { Action } from "@/lib/shared/views/ui/components/context-menu";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

const Spotify: LucideIcon = ({ size, class: className }) => (
	<SpotifyIcon size={size ? Number(size) : 16} class={className as string} />
);

const Youtube: LucideIcon = ({ size, class: className }) => (
	<YoutubeIcon width={size} height={size} class={className as string} />
);

async function copy(link: string) {
	try {
		await Clipboard.SetText(link);
	} catch {
		await navigator.clipboard.writeText(link);
	}
	remember(link);
	notify(t("music.share.copied"));
}

async function youtube(song: Song) {
	const id = song.youtubeId || (await resolveYoutubeId(song));
	if (!id) {
		addError(t("music.share.error"));
		return;
	}
	if (!song.youtubeId) setYoutube(song, id);
	await copy(`https://www.youtube.com/watch?v=${id}`);
}

export function shareActions(song: Song): Action[] {
	const actions: Action[] = [];

	const spotify = spotifyLink(song);
	if (spotify) {
		actions.push({
			icon: Spotify,
			label: t("music.share.spotify"),
			run: () => {
				void copy(spotify).catch((error) => addError(error, "share"));
			},
		});
	}

	if (canShare(song)) {
		actions.push({
			icon: Youtube,
			label: t("music.share.youtube"),
			run: () => {
				void youtube(song).catch((error) => addError(error, "share"));
			},
		});
	}

	return actions;
}

export function shareLink(link: string | null): Action[] {
	if (!link) return [];
	return [
		{
			icon: Link2,
			label: t("music.share.link"),
			run: () => {
				void copy(link).catch((error) => addError(error, "share"));
			},
		},
	];
}

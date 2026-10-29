import { parseSpotifyLink } from "@/lib/music/app/spotify-link";
import { parseYoutubeLink, youtubeList } from "@/lib/music/app/youtube-link";

export type ClipboardLink = {
	kind: "playlist" | "album" | "artist" | "user" | "channel" | "song";
	text: string;
	to: string;
};

// Resolves a pasted link to an in-app route, or to the id GetSong expects for
// a single track. Anything else (plain text, local: ids, other hosts) is null.
export function clipboardLink(text: string): ClipboardLink | null {
	const trimmed = text.trim();
	if (!trimmed) return null;

	const spotify = parseSpotifyLink(trimmed);
	if (spotify) {
		switch (spotify.type) {
			case "track":
				return { kind: "song", text: trimmed, to: spotify.id };
			case "user":
				return {
					kind: "user",
					text: trimmed,
					to: `/spotify/user/${spotify.id}`,
				};
			default:
				return {
					kind: spotify.type,
					text: trimmed,
					to: `/${spotify.type}/${spotify.id}`,
				};
		}
	}

	// A link copied from inside a playlist carries both, and the playlist is what
	// was shared.
	const list = youtubeList(trimmed);
	if (list) {
		return {
			kind: "playlist",
			text: trimmed,
			to: `/playlist/youtube:${list}`,
		};
	}

	const youtube = parseYoutubeLink(trimmed);
	if (youtube) {
		switch (youtube.type) {
			case "video":
				return { kind: "song", text: trimmed, to: `youtube:${youtube.id}` };
			case "playlist":
				return {
					kind: "playlist",
					text: trimmed,
					to: `/playlist/youtube:${youtube.id}`,
				};
			case "channel":
				return {
					kind: "channel",
					text: trimmed,
					to: `/youtube/user/${encodeURIComponent(youtube.id)}`,
				};
		}
	}

	return null;
}

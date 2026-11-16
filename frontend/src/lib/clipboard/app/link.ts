import { parseSpotifyLink } from "@/lib/music/app/spotify-link";
import { parseYoutubeLink, youtubeList } from "@/lib/music/app/youtube-link";
import { parseRoomLink } from "@/lib/sync/app/room-link";

export type ClipboardLink = {
	kind: "playlist" | "album" | "artist" | "user" | "channel" | "song" | "room";
	text: string;
	to: string;
	// room carries what a link has no route for: the code and its password, which
	// are an action (join) rather than a destination.
	room?: { code: string; password: string };
};

// Bare ids and @handles belong to the search box, not the clipboard: a word that
// happens to be eleven characters is not a video, and neither is a mention.
function isLink(text: string): boolean {
	if (text.startsWith("spotify:")) return true;
	try {
		const url = new URL(text.includes("://") ? text : `https://${text}`);
		return url.hostname.includes(".");
	} catch {
		return false;
	}
}

// Resolves a pasted link to an in-app route, or to the id GetSong expects for
// a single track. Anything else (plain text, local: ids, other hosts) is null.
export function clipboardLink(text: string): ClipboardLink | null {
	const trimmed = text.trim();
	if (!trimmed) return null;

	// A room is the app's own link, so it is read before the external ones: it is
	// not a URL anything else would recognise.
	const room = parseRoomLink(trimmed);
	if (room) {
		return {
			kind: "room",
			text: trimmed,
			to: "/rooms",
			room: { code: room.code, password: room.password },
		};
	}

	if (!isLink(trimmed)) return null;

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

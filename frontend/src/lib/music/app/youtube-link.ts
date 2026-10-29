type YouTubeRef = {
	type: "video" | "playlist" | "channel";
	id: string;
};

const VIDEO_ID = /^[A-Za-z0-9_-]{11}$/;
const PLAYLIST_ID = /^(PL|LL|FL|RD|UU|OL|PU)[A-Za-z0-9_-]{10,}$/;
const CHANNEL_ID = /^UC[A-Za-z0-9_-]{22}$/;
const HOSTS = new Set(["youtube.com", "youtu.be", "youtube-nocookie.com"]);
const PATH_TYPES = new Set(["shorts", "embed", "live"]);

function hostOf(url: URL): string {
	return url.hostname.replace(/^(www|music|m)\./, "");
}

function linkURL(query: string): URL | null {
	try {
		return new URL(query.includes("://") ? query : `https://${query}`);
	} catch {
		return null;
	}
}

// The playlist a link carries, even when it also names a video. Pasting a link
// copied from inside a playlist means the playlist: that is what was shared.
export function youtubeList(query: string): string | null {
	const trimmed = query.trim();
	if (!trimmed) return null;

	const url = linkURL(trimmed);
	if (!url || !HOSTS.has(hostOf(url))) return null;

	return url.searchParams.get("list");
}

// Accepts youtube.com / youtu.be links (watch, shorts, embed, live, playlist,
// channel) and bare video, playlist, channel ID or @handle, and returns the
// referenced entity. A video wins over a playlist, so a "song inside a
// playlist" link opens the song.
export function parseYoutubeLink(query: string): YouTubeRef | null {
	const trimmed = query.trim();
	if (!trimmed) return null;

	if (trimmed.startsWith("@")) return { type: "channel", id: trimmed };
	if (CHANNEL_ID.test(trimmed)) return { type: "channel", id: trimmed };
	if (VIDEO_ID.test(trimmed)) return { type: "video", id: trimmed };
	if (PLAYLIST_ID.test(trimmed)) return { type: "playlist", id: trimmed };

	const url = linkURL(trimmed);
	if (!url) return null;
	const host = hostOf(url);
	if (!HOSTS.has(host)) return null;

	const v = url.searchParams.get("v");
	if (v && VIDEO_ID.test(v)) return { type: "video", id: v };

	if (host === "youtu.be") {
		const id = url.pathname.slice(1);
		if (VIDEO_ID.test(id)) return { type: "video", id };
	}

	const [, kind, id] = url.pathname.split("/");
	if (kind?.startsWith("@")) return { type: "channel", id: kind };
	if (kind === "channel" && CHANNEL_ID.test(id ?? "")) {
		return { type: "channel", id: id as string };
	}
	if (kind === "c" || kind === "user") {
		return { type: "channel", id: url.toString() };
	}
	if (PATH_TYPES.has(kind ?? "") && VIDEO_ID.test(id ?? "")) {
		return { type: "video", id: id as string };
	}

	const list = url.searchParams.get("list");
	if (list) return { type: "playlist", id: list };

	return null;
}

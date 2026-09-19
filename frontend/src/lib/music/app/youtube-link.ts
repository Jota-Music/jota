type YouTubeRef = {
	type: "video" | "playlist" | "channel";
	id: string;
};

const videoId = /^[A-Za-z0-9_-]{11}$/;
const playlistId = /^(PL|LL|FL|RD|UU|OL|PU)[A-Za-z0-9_-]{10,}$/;
const channelId = /^UC[A-Za-z0-9_-]{22}$/;
const hosts = new Set(["youtube.com", "youtu.be", "youtube-nocookie.com"]);
const pathTypes = new Set(["shorts", "embed", "live"]);

function hostOf(url: URL): string {
	return url.hostname.replace(/^(www|music|m)\./, "");
}

// Accepts youtube.com / youtu.be links (watch, shorts, embed, live, playlist,
// channel) and bare video, playlist, channel ID or @handle, and returns the
// referenced entity. A video wins over a playlist, so a "song inside a
// playlist" link opens the song.
export function parseYoutubeLink(query: string): YouTubeRef | null {
	const trimmed = query.trim();
	if (!trimmed) return null;

	if (trimmed.startsWith("@")) return { type: "channel", id: trimmed };
	if (channelId.test(trimmed)) return { type: "channel", id: trimmed };
	if (videoId.test(trimmed)) return { type: "video", id: trimmed };
	if (playlistId.test(trimmed)) return { type: "playlist", id: trimmed };

	let url: URL;
	try {
		url = new URL(trimmed.includes("://") ? trimmed : `https://${trimmed}`);
	} catch {
		return null;
	}
	const host = hostOf(url);
	if (!hosts.has(host)) return null;

	const v = url.searchParams.get("v");
	if (v && videoId.test(v)) return { type: "video", id: v };

	if (host === "youtu.be") {
		const id = url.pathname.slice(1);
		if (videoId.test(id)) return { type: "video", id };
	}

	const [, kind, id] = url.pathname.split("/");
	if (kind?.startsWith("@")) return { type: "channel", id: kind };
	if (kind === "channel" && channelId.test(id ?? "")) {
		return { type: "channel", id: id as string };
	}
	if (kind === "c" || kind === "user") {
		return { type: "channel", id: url.toString() };
	}
	if (pathTypes.has(kind ?? "") && videoId.test(id ?? "")) {
		return { type: "video", id: id as string };
	}

	const list = url.searchParams.get("list");
	if (list) return { type: "playlist", id: list };

	return null;
}

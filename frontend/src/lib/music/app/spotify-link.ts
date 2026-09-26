type SpotifyRef = {
	type: "user" | "track" | "album" | "playlist" | "artist";
	id: string;
};

const TYPES = ["user", "track", "album", "playlist", "artist"] as const;
const HOSTS = new Set(["open.spotify.com", "play.spotify.com"]);

// Entity ids are base62 and 21-22 chars long. Anything else never resolves on
// the Spotify side, so a bare name must not parse as a reference. Usernames are
// the exception: they are textual.
const ID_REGEXP = /^[0-9a-zA-Z]{21,22}$/;

function isType(value: string | undefined): value is SpotifyRef["type"] {
	return value !== undefined && (TYPES as readonly string[]).includes(value);
}

function ref(type: string | undefined, id: string): SpotifyRef | null {
	if (!isType(type) || !id) return null;
	if (type !== "user" && !ID_REGEXP.test(id)) return null;
	return { type, id };
}

// Accepts Spotify URIs (spotify:track:<id>) and open.spotify.com links, with
// optional /intl-xx and /embed segments, and returns the referenced entity.
export function parseSpotifyLink(query: string): SpotifyRef | null {
	const trimmed = query.trim();
	if (!trimmed) return null;

	if (trimmed.startsWith("spotify:")) {
		const [scheme, type, ...rest] = trimmed.split(":");
		return scheme === "spotify" ? ref(type, rest.join(":")) : null;
	}

	let url: URL;
	try {
		url = new URL(trimmed.includes("://") ? trimmed : `https://${trimmed}`);
	} catch {
		return null;
	}
	if (!HOSTS.has(url.hostname.replace(/^www\./, ""))) return null;

	const segments = url.pathname.split("/").filter(Boolean);
	if (segments[0]?.startsWith("intl-")) segments.shift();
	if (segments[0] === "embed") segments.shift();

	const [type, id] = segments;
	return ref(type, id ?? "");
}

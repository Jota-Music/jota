export type PlaylistSource = "spotify" | "youtube" | "local";

export function playlistSource(id: string): PlaylistSource {
	if (id.startsWith("local:")) return "local";
	if (id.startsWith("youtube:")) return "youtube";
	if (id.startsWith("spotify:")) return "spotify";
	// Legacy bare Spotify ids.
	return "spotify";
}

export type PlaylistSource = "spotify" | "youtube" | "local";

export function playlistSource(id: string): PlaylistSource {
	if (id.startsWith("local:")) return "local";
	if (id.startsWith("youtube:")) return "youtube";
	return "spotify";
}

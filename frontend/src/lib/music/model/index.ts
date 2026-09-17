import type {
	Album,
	AlbumSummary,
	Artist,
	Audio,
	Follow,
	ArtistDiscography as GeneratedArtistDiscography,
	ArtistInfo as GeneratedArtistInfo,
	Playlist as GeneratedPlaylist,
	PlaylistSummary,
	SearchResult,
	Share,
	Song,
	UserProfile,
} from "@models/music/models";

export type {
	Album,
	AlbumSummary,
	Artist,
	Audio,
	Follow,
	PlaylistSummary,
	SearchResult,
	Share,
	Song,
	UserProfile,
};

// The generated bindings type slices as nullable because Go's JSON
// encoder emits `null` for empty slices. The app only ever consumes these
// as arrays, so narrow them here instead of at every call site.
export type Playlist = Omit<GeneratedPlaylist, "songs"> & { songs: Song[] };
export type ArtistInfo = Omit<GeneratedArtistInfo, "tracks"> & {
	tracks: Song[];
};
export type ArtistDiscography = Omit<GeneratedArtistDiscography, "albums"> & {
	albums: AlbumSummary[];
};

// A control is applied locally and broadcast to the room. Track changes and
// queue edits are not controls: they go through the load round and the shared
// queue respectively.
export type ControlAction =
	| { action: "toggle" }
	| { action: "play" }
	| { action: "pause" }
	| { action: "seek"; positionMs: number }
	| { action: "shuffle"; on: boolean; seed?: number }
	| { action: "repeat" };

export type RepeatMode = "off" | "all" | "one";

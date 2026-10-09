import type {
	Album,
	AlbumSummary,
	Artist,
	Audio,
	ChannelInfo,
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
	ChannelInfo,
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

// A control is applied locally and broadcast to the room. Track changes go
// through the load round; the shared queue travels as a snapshot, except for the
// incremental queue edits below, which ride the control message so a single
// enqueue does not rewrite every member's queue.
export type RepeatMode = "off" | "all" | "one";

export type QueueAction =
	| { action: "enqueue"; songs: Song[]; afterId: string | null }
	| { action: "youtube"; id: string; youtubeId: string };

export type ControlAction =
	| { action: "play" }
	| { action: "pause" }
	| { action: "stop" }
	| { action: "seek"; positionMs: number }
	| { action: "shuffle"; on: boolean; seed?: number }
	| { action: "repeat"; mode: RepeatMode }
	| QueueAction;

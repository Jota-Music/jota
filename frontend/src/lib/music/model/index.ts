interface Share {
	id: string;
	url: string;
}

export interface Album {
	id?: string;
	title: string;
	url: string;
	covers: string[];
}

export interface Artist {
	name: string;
	id?: string;
}

export interface Song {
	id: string;
	url: string;
	name: string;
	duration: number;
	share: Share;
	album: Album;
	artists: Artist[];
	youtubeId?: string;
}

// A control is applied locally and broadcast to the room. Track changes and
// queue edits are not controls: they go through the load round and the shared
// queue respectively.
export type ControlAction =
	| { action: "toggle" }
	| { action: "seek"; positionMs: number }
	| { action: "shuffle" }
	| { action: "repeat" };

export interface Playlist {
	name?: string;
	cover?: string;
	owner?: string;
	songs: Song[];
}

export interface PlaylistSummary {
	id: string;
	name: string;
	mosaic?: string;
	cover?: string;
	subtitle?: string;
	owner?: string;
}

export interface Audio {
	url: string;
	duration: number;
	expireAt: number;
	videoId?: string;
}

export interface AlbumSummary {
	id: string;
	name: string;
	year: number;
	cover?: string;
	group: string;
}

export interface UserProfile {
	displayName: string;
	imageUrl?: string;
}

export interface Follow {
	id: string;
	name: string;
	imageUrl?: string;
	kind: "artist" | "user";
}

export interface ArtistInfo {
	name: string;
	imageUrl?: string;
	tracks: Song[];
}

export interface ArtistDiscography {
	name: string;
	albums: AlbumSummary[];
}

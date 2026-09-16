export interface Share {
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

export interface Page {
	size: number;
	offset: number;
	total: number;
	hasNext: boolean;
}

export interface Playlist {
	name?: string;
	cover?: string;
	songs: Song[];
	page: Page;
}

export interface PlaylistSummary {
	id: string;
	name: string;
	mosaic?: string;
	cover?: string;
	subtitle?: string;
}

export interface Audio {
	url: string;
	duration: number;
	expireAt: number;
	videoId?: string;
	clientName?: string;
}

export interface AlbumSummary {
	id: string;
	name: string;
	year: number;
	cover?: string;
	group: string;
}

export interface UserProfile {
	username: string;
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
	uri: string;
	imageUrl?: string;
	tracks: Song[];
}

export interface ArtistDiscography {
	name: string;
	uri: string;
	albums: AlbumSummary[];
}

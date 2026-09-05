export interface Share {
	id: string;
	url: string;
}

export interface Album {
	title: string;
	url: string;
	covers: string[];
}

export interface Artist {
	name: string;
}

export interface Song {
	id: string;
	name: string;
	duration: number;
	share: Share;
	album: Album;
	artists: Artist[];
	isMock?: boolean;
	youtubeId?: string;
}

export interface Page {
	size: number;
	offset: number;
	total: number;
	hasNext: boolean;
}

export interface Playlist {
	songs: Song[];
	page: Page;
}

export interface Audio {
	url: string;
	duration: number;
	ttl: number;
	youtube: string;
}

export interface AlbumSummary {
	id: string;
	name: string;
	year: number;
	cover?: string;
	group: string;
}

export interface ArtistInfo {
	name: string;
	uri: string;
	tracks: Song[];
}

export interface ArtistDiscography {
	name: string;
	uri: string;
	albums: AlbumSummary[];
}

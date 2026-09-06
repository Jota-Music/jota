export namespace app {
	
	export class SpotifyStatus {
	    connected: boolean;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new SpotifyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.user = source["user"];
	    }
	}

}

export namespace music {
	
	export class Album {
	    title: string;
	    url: string;
	    covers: string[];
	
	    static createFrom(source: any = {}) {
	        return new Album(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.covers = source["covers"];
	    }
	}
	export class AlbumSummary {
	    id: string;
	    name: string;
	    year: number;
	    cover?: string;
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new AlbumSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.year = source["year"];
	        this.cover = source["cover"];
	        this.group = source["group"];
	    }
	}
	export class Artist {
	    id?: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Artist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class ArtistDiscography {
	    name: string;
	    uri: string;
	    albums: AlbumSummary[];
	
	    static createFrom(source: any = {}) {
	        return new ArtistDiscography(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.uri = source["uri"];
	        this.albums = this.convertValues(source["albums"], AlbumSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Share {
	    id: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new Share(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	    }
	}
	export class Song {
	    id: string;
	    url: string;
	    name: string;
	    duration: number;
	    share: Share;
	    album: Album;
	    artists: Artist[];
	
	    static createFrom(source: any = {}) {
	        return new Song(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.name = source["name"];
	        this.duration = source["duration"];
	        this.share = this.convertValues(source["share"], Share);
	        this.album = this.convertValues(source["album"], Album);
	        this.artists = this.convertValues(source["artists"], Artist);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ArtistInfo {
	    name: string;
	    uri: string;
	    imageUrl?: string;
	    tracks: Song[];
	
	    static createFrom(source: any = {}) {
	        return new ArtistInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.uri = source["uri"];
	        this.imageUrl = source["imageUrl"];
	        this.tracks = this.convertValues(source["tracks"], Song);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Page {
	    size: number;
	    offset: number;
	    total: number;
	    hasNext: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Page(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.size = source["size"];
	        this.offset = source["offset"];
	        this.total = source["total"];
	        this.hasNext = source["hasNext"];
	    }
	}
	export class Playlist {
	    songs: Song[];
	    page: Page;
	
	    static createFrom(source: any = {}) {
	        return new Playlist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.songs = this.convertValues(source["songs"], Song);
	        this.page = this.convertValues(source["page"], Page);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlaylistSummary {
	    id: string;
	    name: string;
	    mosaic?: string;
	    cover?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.mosaic = source["mosaic"];
	        this.cover = source["cover"];
	    }
	}
	export class SearchResult {
	    uri: string;
	    name: string;
	    type: string;
	    coverUrl?: string;
	    artists?: string[];
	    ownerName?: string;
	    trackCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uri = source["uri"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.coverUrl = source["coverUrl"];
	        this.artists = source["artists"];
	        this.ownerName = source["ownerName"];
	        this.trackCount = source["trackCount"];
	    }
	}
	

}

export namespace youtube {
	
	export class Audio {
	    url: string;
	    duration: number;
	    expireAt: number;
	    videoId: string;
	
	    static createFrom(source: any = {}) {
	        return new Audio(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.duration = source["duration"];
	        this.expireAt = source["expireAt"];
	        this.videoId = source["videoId"];
	    }
	}
	export class Video {
	    ID: string;
	    Title: string;
	
	    static createFrom(source: any = {}) {
	        return new Video(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Title = source["Title"];
	    }
	}

}


import { getAudio } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";

type CachedAudio = {
	url: string;
	audio?: HTMLAudioElement;
	lastUsed: number;
	youtube: string;
};

// biome-ignore lint/complexity/noStaticOnlyClass: <   >
export class AudioCache {
	private static cache = new Map<string, CachedAudio>();
	private static pending = new Map<string, Promise<CachedAudio>>();

	private static maxCache = 25;

	/* -------------------------------------------------
       INTERNAL HELPERS
    -------------------------------------------------- */

	private static touch(id: string) {
		const item = AudioCache.cache.get(id);
		if (!item) return;

		item.lastUsed = Date.now();

		// LRU: reinsert to mark as most recently used
		AudioCache.cache.delete(id);
		AudioCache.cache.set(id, item);
	}

	private static release(item: CachedAudio) {
		if (item.audio) {
			item.audio.pause();
			item.audio.src = "";
			item.audio.load();
		}
	}

	private static enforceLimit() {
		if (AudioCache.cache.size <= AudioCache.maxCache) return;

		const entries = [...AudioCache.cache.entries()];

		// sort by least recently used
		entries.sort((a, b) => a[1].lastUsed - b[1].lastUsed);

		const excess = AudioCache.cache.size - AudioCache.maxCache;

		for (let i = 0; i < excess; i++) {
			const [id, item] = entries[i];

			AudioCache.release(item);
			AudioCache.cache.delete(id);
		}
	}

	/* -------------------------------------------------
       GET (CACHE + DEDUPE)
    -------------------------------------------------- */

	static async get(song: Song): Promise<CachedAudio> {
		const cached = AudioCache.cache.get(song.id);
		if (cached) {
			AudioCache.touch(song.id);
			return cached;
		}

		const existing = AudioCache.pending.get(song.id);
		if (existing) return existing;

		const request = getAudio(song)
			.then((data) => {
				const value: CachedAudio = {
					url: data.url,
					lastUsed: Date.now(),
					youtube: data.youtube,
				};

				AudioCache.cache.set(song.id, value);
				AudioCache.enforceLimit();

				return value;
			})
			.finally(() => {
				AudioCache.pending.delete(song.id);
			});

		AudioCache.pending.set(song.id, request);

		return request;
	}

	/* -------------------------------------------------
       PRELOAD AUDIO
    -------------------------------------------------- */

	static async preload(...songs: Song[]): Promise<void> {
		await Promise.all(
			songs.map(async (song) => {
				if (AudioCache.cache.has(song.id) || AudioCache.pending.has(song.id))
					return;

				const data = await AudioCache.get(song);

				if (data.audio) return;

				const audio = new Audio(data.url);
				audio.preload = "auto";
				audio.load();

				const cached = AudioCache.cache.get(song.id);
				if (cached) {
					cached.audio = audio;
					cached.lastUsed = Date.now();
				}
			}),
		);
	}

	/* -------------------------------------------------
       GET AUDIO ELEMENT
    -------------------------------------------------- */

	static async getAudioElement(song: Song): Promise<HTMLAudioElement> {
		const cached = await AudioCache.get(song);

		if (cached.audio) {
			return cached.audio;
		}

		const audio = new Audio(cached.url);
		audio.preload = "auto";
		audio.load();

		cached.audio = audio;
		cached.lastUsed = Date.now();

		return audio;
	}

	/* -------------------------------------------------
       CACHE MANAGEMENT
    -------------------------------------------------- */

	static clear(): void {
		for (const [, item] of AudioCache.cache) {
			AudioCache.release(item);
		}

		AudioCache.cache.clear();
		AudioCache.pending.clear();
	}

	static remove(id: string): void {
		const item = AudioCache.cache.get(id);
		if (item) AudioCache.release(item);

		AudioCache.cache.delete(id);
		AudioCache.pending.delete(id);
	}

	/* -------------------------------------------------
       DEBUG
    -------------------------------------------------- */

	static has(id: string): boolean {
		return AudioCache.cache.has(id);
	}

	static size(): number {
		return AudioCache.cache.size;
	}
}

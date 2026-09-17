import { getAudio } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";

type CachedAudio = {
	url: string;
	duration: number;
	lastUsed: number;
	youtube: string;
	expireAt: number;
};

// WebKit grants autoplay per media element, so a new element created mid-playlist
// needs its own user gesture and gets blocked. Reuse one element for every track
// and swap its src, which is WebKit's documented way to play tracks back to back.
// biome-ignore lint/complexity/noStaticOnlyClass: <   >
export class AudioCache {
	private static cache = new Map<string, CachedAudio>();
	private static pending = new Map<string, Promise<CachedAudio>>();

	private static maxCache = 25;

	private static expiryMarginSeconds = 30;

	// Bound a stuck resolve: a promise that never settles must not wedge every
	// later track behind it (the remote playback tail awaits this).
	private static resolveTimeoutMs = 15000;

	private static withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
		return new Promise<T>((resolve, reject) => {
			const timer = setTimeout(
				() => reject(new Error("audio resolve timed out")),
				ms,
			);
			promise.then(
				(value) => {
					clearTimeout(timer);
					resolve(value);
				},
				(error) => {
					clearTimeout(timer);
					reject(error);
				},
			);
		});
	}

	private static player: HTMLAudioElement | null = null;
	private static playerUrl: string | null = null;

	// Stream URLs carry a YouTube expiry; a cached URL past it must be re-resolved.
	private static isExpired(item: CachedAudio): boolean {
		if (!item.expireAt) return false;
		const now = Math.floor(Date.now() / 1000);
		return item.expireAt <= now + AudioCache.expiryMarginSeconds;
	}

	private static touch(id: string) {
		const item = AudioCache.cache.get(id);
		if (!item) return;

		item.lastUsed = Date.now();

		// LRU: reinsert to mark as most recently used
		AudioCache.cache.delete(id);
		AudioCache.cache.set(id, item);
	}

	private static enforceLimit() {
		if (AudioCache.cache.size <= AudioCache.maxCache) return;

		const entries = [...AudioCache.cache.entries()];

		// sort by least recently used
		entries.sort((a, b) => a[1].lastUsed - b[1].lastUsed);

		const excess = AudioCache.cache.size - AudioCache.maxCache;

		for (let i = 0; i < excess; i++) {
			AudioCache.cache.delete(entries[i][0]);
		}
	}

	static async get(song: Song): Promise<CachedAudio> {
		const cached = AudioCache.cache.get(song.id);
		if (cached) {
			if (AudioCache.isExpired(cached)) {
				AudioCache.remove(song.id);
			} else {
				AudioCache.touch(song.id);
				return cached;
			}
		}

		const existing = AudioCache.pending.get(song.id);
		if (existing) return existing;

		const request = AudioCache.withTimeout(
			getAudio(song),
			AudioCache.resolveTimeoutMs,
		)
			.then((data) => {
				const value: CachedAudio = {
					url: data.url,
					duration: data.duration,
					lastUsed: Date.now(),
					youtube: data.youtube,
					expireAt: data.expireAt ?? 0,
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

	// Only the resolved URL is worth warming now that playback shares one element.
	static async preload(...songs: Song[]): Promise<void> {
		await Promise.all(
			songs.map((song) => AudioCache.get(song).catch(() => undefined)),
		);
	}

	// The single element every track plays through, so its autoplay grant
	// survives the whole session instead of being needed per song.
	static async getAudioElement(song: Song): Promise<HTMLAudioElement> {
		const cached = await AudioCache.get(song);

		if (!AudioCache.player) {
			const player = new Audio();
			player.preload = "auto";
			AudioCache.player = player;
		}
		const player = AudioCache.player;

		if (AudioCache.playerUrl !== cached.url || player.error) {
			AudioCache.playerUrl = cached.url;
			player.src = cached.url;
			player.load();
		}

		return player;
	}

	// Keep the element; just make the next getAudioElement reload the stream.
	static releaseElement(_id: string): void {
		AudioCache.player?.pause();
		AudioCache.playerUrl = null;
	}

	static remove(id: string): void {
		AudioCache.cache.delete(id);
		AudioCache.pending.delete(id);
		AudioCache.playerUrl = null;
	}
}

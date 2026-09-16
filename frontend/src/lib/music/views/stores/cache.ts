import { getAudio } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";

type CachedAudio = {
	url: string;
	duration: number;
	audio?: HTMLAudioElement;
	lastUsed: number;
	youtube: string;
	expireAt: number;
};

// biome-ignore lint/complexity/noStaticOnlyClass: <   >
export class AudioCache {
	private static cache = new Map<string, CachedAudio>();
	private static pending = new Map<string, Promise<CachedAudio>>();

	private static maxCache = 25;

	private static maxElements = 3;

	private static pinnedId: string | null = null;

	private static expiryMarginSeconds = 30;

	// Stream URLs carry a YouTube expiry; a cached URL past it must be re-resolved.
	private static isExpired(item: CachedAudio): boolean {
		if (!item.expireAt) return false;
		const now = Math.floor(Date.now() / 1000);
		return item.expireAt <= now + AudioCache.expiryMarginSeconds;
	}

	// An element whose load failed stays dead until rebuilt, so never reuse it.
	private static isBroken(audio: HTMLAudioElement | undefined): boolean {
		return (
			!!audio &&
			(audio.error !== null ||
				audio.networkState === HTMLMediaElement.NETWORK_NO_SOURCE)
		);
	}

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
		if (!item.audio) return;

		item.audio.pause();
		item.audio.src = "";
		item.audio.load();
		item.audio = undefined;
	}

	// Pin the audio element the player is using so eviction never cuts playback.
	static pin(id: string | null) {
		AudioCache.pinnedId = id;
	}

	// Keep only the most recent audio elements alive. URLs stay cached, so an
	// evicted song is re-created on demand without resolving the stream again.
	private static enforceElements() {
		const pinned = AudioCache.pinnedId
			? AudioCache.cache.get(AudioCache.pinnedId)
			: undefined;

		const droppable = [...AudioCache.cache.values()].filter(
			(item) => item.audio && item !== pinned,
		);

		const excess =
			droppable.length - (AudioCache.maxElements - (pinned ? 1 : 0));

		for (let i = 0; i < excess; i++) {
			AudioCache.release(droppable[i]);
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
			if (AudioCache.isExpired(cached)) {
				AudioCache.remove(song.id);
			} else {
				AudioCache.touch(song.id);
				return cached;
			}
		}

		const existing = AudioCache.pending.get(song.id);
		if (existing) return existing;

		const request = getAudio(song)
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

	/* -------------------------------------------------
       PRELOAD AUDIO
    -------------------------------------------------- */

	static async preload(...songs: Song[]): Promise<void> {
		await Promise.all(
			songs.map(async (song) => {
				if (AudioCache.cache.has(song.id) || AudioCache.pending.has(song.id))
					return;

				const data = await AudioCache.get(song);

				if (data.audio && !AudioCache.isBroken(data.audio)) return;
				if (data.audio) AudioCache.release(data);

				const audio = new Audio(data.url);
				audio.preload = "auto";
				audio.load();

				const cached = AudioCache.cache.get(song.id);
				if (cached) {
					cached.audio = audio;
					cached.lastUsed = Date.now();
					AudioCache.touch(song.id);
					AudioCache.enforceElements();
				}
			}),
		);
	}

	/* -------------------------------------------------
       GET AUDIO ELEMENT
    -------------------------------------------------- */

	static async getAudioElement(song: Song): Promise<HTMLAudioElement> {
		const cached = await AudioCache.get(song);

		if (cached.audio && !AudioCache.isBroken(cached.audio)) {
			AudioCache.touch(song.id);
			return cached.audio;
		}
		if (cached.audio) AudioCache.release(cached);

		const audio = new Audio(cached.url);
		audio.preload = "auto";
		audio.load();

		cached.audio = audio;
		cached.lastUsed = Date.now();
		AudioCache.enforceElements();

		return audio;
	}

	/* -------------------------------------------------
       CACHE MANAGEMENT
    -------------------------------------------------- */

	static remove(id: string): void {
		const item = AudioCache.cache.get(id);
		if (item) AudioCache.release(item);

		AudioCache.cache.delete(id);
		AudioCache.pending.delete(id);
	}

	// Drop a cached audio element that the player has emptied (src=""), so a
	// later replay rebuilds it instead of reusing a src-less element.
	static releaseElement(id: string): void {
		const item = AudioCache.cache.get(id);
		if (item) AudioCache.release(item);
	}
}

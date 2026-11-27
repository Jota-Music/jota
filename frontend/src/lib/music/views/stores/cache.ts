import { getAudio } from "@/lib/music/app/get-audio";
import type { Song } from "@/lib/music/model";

type CachedAudio = {
	url: string;
	duration: number;
	lastUsed: number;
	youtube: string;
	expireAt: number;
	mime: string;
};

// The stream is fed to the player through a MediaSource instead of its URL.
// WebKit buffers a URL stream on disk and only calls the buffering done once the
// whole file is there, so playback pauses about a second in; MSE buffers in
// memory and starts as soon as the bytes are appended. Bounded so a long track
// does not sit in memory and a stuck fetch falls back to streaming.
const MSE_LIMIT = 32 * 1024 * 1024;
const MSE_TIMEOUT_MS = 15000;

function streamMime(url: string): string {
	try {
		return new URL(url).searchParams.get("mime") ?? "";
	} catch {
		return "";
	}
}

// A 45-byte silent 8kHz WAV: played inside the first user gesture to grant the
// shared player element autoplay for the whole session.
export const SILENT_WAV =
	"data:audio/wav;base64,UklGRiUAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQEAAAAA";

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
	// later track behind it (the remote playback tail awaits this). Generous so
	// a slow or throttled connection still has room to land a stream.
	private static resolveTimeoutMs = 25000;

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
	private static mediaUrl: string | null = null;
	private static mediaFor: string | null = null;
	private static mediaGen = 0;

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
					mime: streamMime(data.url),
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

	// Perfectly silent clip played inside the first user gesture. WebKit grants
	// autoplay per media element, so grabbing that grant on the shared player
	// while the gesture is live lets every later track start without another
	// click, even when the play call lands well past the activation window.
	static primeAutoplay(): void {
		const player = AudioCache.player ?? new Audio();
		AudioCache.player = player;
		player.muted = true;
		player.preload = "auto";
		if (AudioCache.playerUrl === SILENT_WAV) return;
		AudioCache.playerUrl = SILENT_WAV;
		player.src = SILENT_WAV;
		player.load();
		void player.play().then(
			() => player.pause(),
			() => {},
		);
	}

	// The single element every track plays through, so its autoplay grant
	// survives the whole session instead of being needed per song.
	static async getAudioElement(
		song: Song,
		preload?: "auto" | "metadata" | "none",
	): Promise<HTMLAudioElement> {
		const cached = await AudioCache.get(song);

		if (!AudioCache.player) {
			const player = new Audio();
			player.preload = "auto";
			AudioCache.player = player;
		}
		const player = AudioCache.player;

		if (AudioCache.playerUrl !== cached.url || player.error || player.ended) {
			AudioCache.playerUrl = cached.url;
			if (preload) player.preload = preload;
			await AudioCache.attach(player, cached);
		}

		return player;
	}

	private static async attach(
		player: HTMLAudioElement,
		cached: CachedAudio,
	): Promise<void> {
		// The autoplay prime swaps its silent clip into the element, and the
		// source this track already appended is still open: reattach it instead
		// of downloading the whole track again.
		if (AudioCache.mediaFor === cached.url && AudioCache.mediaUrl) {
			player.src = AudioCache.mediaUrl;
			player.load();
			return;
		}
		if (!(await AudioCache.attachMedia(player, cached))) {
			AudioCache.revokeMedia();
			player.src = cached.url;
			player.load();
		}
	}

	private static async attachMedia(
		player: HTMLAudioElement,
		cached: CachedAudio,
	): Promise<boolean> {
		if (
			!cached.mime ||
			typeof MediaSource === "undefined" ||
			!MediaSource.isTypeSupported(cached.mime)
		) {
			return false;
		}
		const controller = new AbortController();
		const timer = setTimeout(() => controller.abort(), MSE_TIMEOUT_MS);
		try {
			const response = await fetch(cached.url, { signal: controller.signal });
			const length = Number(response.headers.get("Content-Length") ?? 0);
			if (!response.ok || length > MSE_LIMIT) return false;
			const reader = response.body?.getReader();
			if (!reader) return false;
			const first = await reader.read();
			if (first.done || !first.value) return false;
			const media = new MediaSource();
			AudioCache.revokeMedia();
			AudioCache.mediaUrl = URL.createObjectURL(media);
			AudioCache.mediaFor = cached.url;
			player.src = AudioCache.mediaUrl;
			player.load();
			await openMedia(media);
			const buffer = media.addSourceBuffer(cached.mime);
			await append(buffer, first.value.slice().buffer);
			// The rest keeps landing while the track plays: the first piece is
			// enough to start, so a slow link does not wait for the whole file.
			void AudioCache.fill(AudioCache.mediaGen, reader, buffer);
			return true;
		} catch {
			return false;
		} finally {
			clearTimeout(timer);
		}
	}

	private static async fill(
		gen: number,
		reader: ReadableStreamDefaultReader<Uint8Array>,
		buffer: SourceBuffer,
	): Promise<void> {
		try {
			while (gen === AudioCache.mediaGen) {
				const next = await reader.read();
				if (next.done || !next.value) break;
				await append(buffer, next.value.slice().buffer);
			}
			if (gen !== AudioCache.mediaGen) void reader.cancel();
		} catch {}
	}

	private static revokeMedia(): void {
		AudioCache.mediaGen += 1;
		if (AudioCache.mediaUrl) URL.revokeObjectURL(AudioCache.mediaUrl);
		AudioCache.mediaUrl = null;
		AudioCache.mediaFor = null;
	}

	// Whether this song is playing from an attached MediaSource rather than the
	// element's URL. The start of an MSE source has no WebKit hiccup to hide.
	static streamed(id: string): boolean {
		const cached = AudioCache.cache.get(id);
		return (
			!!cached &&
			AudioCache.mediaFor === cached.url &&
			AudioCache.mediaUrl !== null
		);
	}

	// Whether the shared element still holds the stream resolved for this song.
	// The silent autoplay prime swaps its src behind the player's back, and a
	// load that trusts the stale element ends up playing that clip.
	static holds(id: string): boolean {
		const cached = AudioCache.cache.get(id);
		return !!cached && cached.url === AudioCache.playerUrl;
	}

	// Keep the element; just make the next getAudioElement reload the stream.
	static releaseElement(_id: string): void {
		AudioCache.player?.pause();
		AudioCache.playerUrl = null;
		AudioCache.revokeMedia();
	}

	// Detach the stream currently loaded in the shared element so WebKit can
	// free its buffered data while idle; the next getAudioElement re-attaches.
	static dropBuffers(): void {
		const player = AudioCache.player;
		if (!player) return;
		player.pause();
		player.removeAttribute("src");
		player.load();
		AudioCache.playerUrl = null;
		AudioCache.revokeMedia();
	}

	static remove(id: string): void {
		AudioCache.cache.delete(id);
		AudioCache.pending.delete(id);
		AudioCache.playerUrl = null;
		AudioCache.revokeMedia();
	}
}

function openMedia(media: MediaSource): Promise<void> {
	if (media.readyState === "open") return Promise.resolve();
	return new Promise((resolve, reject) => {
		const timer = setTimeout(
			() => reject(new Error("media source never opened")),
			MSE_TIMEOUT_MS,
		);
		media.addEventListener(
			"sourceopen",
			() => {
				clearTimeout(timer);
				resolve();
			},
			{ once: true },
		);
	});
}

function append(buffer: SourceBuffer, data: ArrayBuffer): Promise<void> {
	return new Promise((resolve, reject) => {
		const timer = setTimeout(
			() => reject(new Error("append never finished")),
			MSE_TIMEOUT_MS,
		);
		const done = (err?: Error) => {
			clearTimeout(timer);
			if (err) reject(err);
			else resolve();
		};
		buffer.addEventListener("updateend", () => done(), { once: true });
		buffer.addEventListener("error", () => done(new Error("append failed")), {
			once: true,
		});
		try {
			buffer.appendBuffer(data);
		} catch (err) {
			done(err instanceof Error ? err : new Error("append failed"));
		}
	});
}

import { SetYouTubeId } from "@bindings/app";
import { effect } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import {
	currentSong,
	getPlaybackSeconds,
	ignoreTabMute,
	isPlaying,
	play,
	seek,
	togglePlayPause,
} from "@/lib/music/views/stores/audio";
import { autoAdvance } from "@/lib/music/views/stores/player";
import {
	currentIndex,
	persistQueue,
	queue,
} from "@/lib/music/views/stores/queue";
import { binaryColor, dominantColor } from "@/lib/music/views/stores/theme";
import * as transport from "@/lib/sync/app/transport";
import type { PeerMessage } from "@/lib/sync/model";
import * as store from "@/lib/sync/views/stores";

const samples: number[] = [];
let pingId = 0;
const pending = new Map<number, number>();

effect(() => {
	if (store.role.value !== "host" || store.status.value !== "open") return;
	queue.value;
	sendQueue();
});

effect(() => {
	if (store.role.value !== "host" || store.status.value !== "open") return;
	currentSong.value;
	isPlaying.value;
	currentIndex.value;
	transport.send({
		t: "state",
		at: serverNow(),
		playing: isPlaying.value,
		positionMs: Math.round(getPlaybackSeconds() * 1000),
		songId: currentSong.value?.id ?? "",
		youtubeId: currentSong.value?.youtubeId,
		index: currentIndex.value,
		color: dominantColor.value,
		binary: binaryColor.value,
	});
});

setInterval(() => {
	if (store.role.value !== "host" || store.status.value !== "open") return;
	transport.send({
		t: "heartbeat",
		at: serverNow(),
		playing: isPlaying.value,
		positionMs: Math.round(getPlaybackSeconds() * 1000),
		songId: currentSong.value?.id ?? "",
		color: dominantColor.value,
		binary: binaryColor.value,
	});
}, 2000);

effect(() => {
	if (store.role.value === "off" || store.status.value !== "open") return;
	samples.length = 0;
	pending.clear();
	const interval = window.setInterval(() => {
		const id = ++pingId;
		const at = Date.now();
		pending.set(id, at);
		transport.send({ t: "ping", id, at });
		for (const [key, sentAt] of pending) {
			if (at - sentAt > 5000) pending.delete(key);
		}
	}, 1000);
	return () => window.clearInterval(interval);
});

effect(() => {
	const raw = store.lastMessage.value;
	if (!raw) return;
	let msg: PeerMessage;
	try {
		msg = JSON.parse(raw) as PeerMessage;
	} catch {
		console.error("sync: invalid message");
		return;
	}

	if (msg.t === "role") {
		store.role.value = msg.role;
		return;
	}
	if (store.role.value === "off") return;

	switch (msg.t) {
		case "pong":
			applyPong(msg);
			break;
		case "members":
			store.peers.value = msg.count;
			break;
		case "error":
			store.error.value = msg.reason;
			break;
		default:
			if (store.role.value === "guest") handleGuest(msg);
	}
});

effect(() => {
	if (store.role.value === "guest") {
		autoAdvance.value = false;
		ignoreTabMute.value = true;
	} else {
		if (!autoAdvance.value) autoAdvance.value = true;
		if (ignoreTabMute.value) ignoreTabMute.value = false;
	}
});

function sendQueue(): void {
	transport.send({ t: "queue", data: JSON.stringify(queue.value) });
}

function handleGuest(msg: PeerMessage): void {
	switch (msg.t) {
		case "queue":
			try {
				queue.value = JSON.parse(msg.data) as Song[];
				persistQueue();
			} catch {
				console.error("sync: invalid queue");
			}
			break;
		case "state":
			void applyState(msg);
			break;
		case "heartbeat":
			applyHeartbeat(msg);
			break;
	}
}

async function applyState(
	m: Extract<PeerMessage, { t: "state" }>,
): Promise<void> {
	if (m.youtubeId && m.songId) {
		try {
			await SetYouTubeId(m.songId, m.youtubeId);
		} catch {
			// keep going
		}
	}

	const target = projected(m.positionMs, m.at);
	const sameSong = currentSong.value?.id === m.songId;

	currentIndex.value = m.index;
	applyColor(m.color, m.binary);

	if (!sameSong) {
		const song = queue.value[m.index];
		if (song) await play(song, target);
		return;
	}

	if (m.playing && !isPlaying.value) {
		await togglePlayPause();
	} else if (!m.playing && isPlaying.value) {
		await togglePlayPause();
	}
	seek(target);
}

function applyHeartbeat(m: Extract<PeerMessage, { t: "heartbeat" }>): void {
	if (currentSong.value?.id !== m.songId) return;
	applyColor(m.color, m.binary);
	const target = projected(m.positionMs, m.at);

	if (m.playing !== isPlaying.value) {
		void togglePlayPause();
		return;
	}
	if (Math.abs(getPlaybackSeconds() - target) > 0.5) {
		seek(target);
	}
}

function applyPong(m: Extract<PeerMessage, { t: "pong" }>): void {
	const sent = pending.get(m.id);
	if (sent == null) return;
	pending.delete(m.id);
	const now = Date.now();
	const rtt = now - sent;
	const sample = m.echo - sent - rtt / 2;
	samples.push(sample);
	store.offsetMs.value = median(samples);
}

function median(values: number[]): number {
	if (values.length === 0) return 0;
	const sorted = [...values].sort((a, b) => a - b);
	const mid = Math.floor(sorted.length / 2);
	return sorted.length % 2 !== 0
		? sorted[mid]
		: (sorted[mid - 1] + sorted[mid]) / 2;
}

function serverNow(): number {
	return Date.now() + store.offsetMs.value;
}

function projected(positionMs: number, sentAt: number): number {
	return (positionMs + (serverNow() - sentAt)) / 1000;
}

function applyColor(color?: string | null, binary?: string | null): void {
	if (color) {
		dominantColor.value = color;
		document.documentElement.style.setProperty("--dominant-color", color);
	}
	if (binary) {
		binaryColor.value = binary;
		document.documentElement.style.setProperty("--binary-color", binary);
	}
}

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
import * as transport from "@/lib/p2p/app/transport";
import type { PeerMessage } from "@/lib/p2p/model";
import * as store from "@/lib/p2p/views/stores";

const QUEUE_CHUNK = 64 * 1024;

const samples: number[] = [];
let pingId = 0;
let queueTransferId = 0;
let queueReady = false;
const pending = new Map<number, number>();
const queueBuffers = new Map<number, (string | undefined)[]>();
let pendingState: Extract<PeerMessage, { t: "state" }> | null = null;

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
		at: Date.now(),
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
		at: Date.now(),
		playing: isPlaying.value,
		positionMs: Math.round(getPlaybackSeconds() * 1000),
		songId: currentSong.value?.id ?? "",
		color: dominantColor.value,
		binary: binaryColor.value,
	});
}, 2000);

effect(() => {
	const raw = store.lastMessage.value;
	if (!raw || store.role.value !== "guest") return;
	try {
		const msg = JSON.parse(raw) as PeerMessage;
		handleGuest(msg);
	} catch {
		console.error("p2p: invalid message");
	}
});

effect(() => {
	const raw = store.lastMessage.value;
	if (!raw || store.role.value !== "host") return;
	try {
		const msg = JSON.parse(raw) as PeerMessage;
		if (msg.t === "ping") {
			transport.send({
				t: "pong",
				id: msg.id,
				at: msg.at,
				echo: Date.now(),
			});
		}
	} catch {
		console.error("p2p: invalid message");
	}
});

effect(() => {
	if (store.role.value !== "guest" || store.status.value !== "open") {
		return;
	}
	pendingState = null;
	queueReady = false;
	queueBuffers.clear();
	samples.length = 0;
	pending.clear();
	const interval = window.setInterval(() => {
		const id = ++pingId;
		const at = Date.now();
		pending.set(id, at);
		transport.send({ t: "ping", id, at });
	}, 1000);
	window.setTimeout(() => window.clearInterval(interval), 5500);
	return () => window.clearInterval(interval);
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
	const json = JSON.stringify(queue.value);
	const id = ++queueTransferId;
	const n = Math.max(1, Math.ceil(json.length / QUEUE_CHUNK));
	for (let i = 0; i < n; i++) {
		transport.send({
			t: "queue",
			id,
			i,
			n,
			data: json.slice(i * QUEUE_CHUNK, (i + 1) * QUEUE_CHUNK),
		});
	}
}

function handleGuest(msg: PeerMessage): void {
	switch (msg.t) {
		case "queue":
			applyQueue(msg);
			break;
		case "state":
			if (!queueReady) {
				pendingState = msg;
			} else {
				void applyState(msg);
			}
			break;
		case "heartbeat":
			applyHeartbeat(msg);
			break;
		case "pong":
			applyPong(msg);
			break;
	}
}

function applyQueue(m: Extract<PeerMessage, { t: "queue" }>): void {
	if (m.i === 0) queueBuffers.clear();

	let parts = queueBuffers.get(m.id);
	if (!parts) {
		parts = new Array<string | undefined>(m.n);
		queueBuffers.set(m.id, parts);
	}
	parts[m.i] = m.data;
	if (parts.includes(undefined)) return;

	queueBuffers.delete(m.id);
	try {
		queue.value = JSON.parse((parts as string[]).join("")) as Song[];
		persistQueue();
		queueReady = true;
	} catch {
		console.error("p2p: invalid queue");
		return;
	}
	if (pendingState) {
		const ps = pendingState;
		pendingState = null;
		void applyState(ps);
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

function projected(positionMs: number, sentAt: number): number {
	return (positionMs + (Date.now() - sentAt) + store.offsetMs.value) / 1000;
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

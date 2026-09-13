import { SetYouTubeId } from "@bindings/app";
import { effect } from "@preact/signals";
import type { Song } from "@/lib/music/model";
import {
	currentSong,
	getPlaybackSeconds,
	ignoreTabMute,
	isLoading,
	isPlaying,
	play,
	prepareSong,
	seek,
	seekCount,
	setPlaybackRate,
	togglePlayPause,
	warm,
} from "@/lib/music/views/stores/audio";
import {
	autoAdvance,
	enqueue,
	moveAfterCurrent,
	moveQueue,
	nextSong,
	playAt,
	playFromQueueSelection,
	preloadUpcomingSongs,
	prevSong,
	seekFromLocalControl,
	unqueue,
} from "@/lib/music/views/stores/player";
import {
	currentIndex,
	cycleRepeat,
	persistQueue,
	queue,
	repeat,
	setRepeat,
	setShuffle,
	shuffle,
	toggleShuffle,
} from "@/lib/music/views/stores/queue";
import { playGate, remoteControl } from "@/lib/music/views/stores/remote";
import { binaryColor, dominantColor } from "@/lib/music/views/stores/theme";
import * as transport from "@/lib/sync/app/transport";
import type { ControlAction, PeerMessage } from "@/lib/sync/model";
import * as store from "@/lib/sync/views/stores";

const samples: number[] = [];
let pingId = 0;
let controlAt = 0;
let gateState: {
	songId: string;
	needed: number;
	count: number;
	resolve: () => void;
	timer: number;
} | null = null;
const pending = new Map<number, number>();

effect(() => {
	if (store.role.value !== "host" || store.status.value !== "open") return;
	queue.value;
	sendQueue();
});

let lastStateAt = 0;
let stateTimer = 0;

function sendState(): void {
	lastStateAt = Date.now();
	transport.send({
		t: "state",
		at: serverNow(),
		playing: isPlaying.value,
		positionMs: Math.round(getPlaybackSeconds() * 1000),
		songId: currentSong.value?.id ?? "",
		youtubeId: currentSong.value?.youtubeId,
		index: currentIndex.value,
		shuffle: shuffle.value,
		repeat: repeat.value,
		color: dominantColor.value,
		binary: binaryColor.value,
	});
}

effect(() => {
	if (store.role.value !== "host" || store.status.value !== "open") return;
	seekCount.value;
	shuffle.value;
	repeat.value;
	currentSong.value;
	isPlaying.value;
	currentIndex.value;
	dominantColor.value;
	binaryColor.value;
	const wait = 150 - (Date.now() - lastStateAt);
	if (wait <= 0) {
		sendState();
	} else if (!stateTimer) {
		stateTimer = window.setTimeout(() => {
			stateTimer = 0;
			sendState();
		}, wait);
	}
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
	if (msg.t === "error") {
		store.error.value = msg.reason;
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
		case "control":
			if (store.role.value === "host") applyControl(msg);
			break;
		case "ready":
			if (store.role.value === "host") markReady(msg.songId);
			break;
		case "prepare":
			if (store.role.value === "guest") void warmRemote(msg);
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

effect(() => {
	if (store.role.value !== "guest" || store.status.value !== "open") return;
	preloadUpcomingSongs(queue.value, currentIndex.value);
});

effect(() => {
	remoteControl.value = store.role.value === "guest" ? sendControl : null;
	playGate.value =
		store.role.value === "host" && store.status.value === "open" ? gate : null;
});

function sendControl(a: ControlAction): void {
	controlAt = serverNow() + 250;
	transport.send({ t: "control", ...a });
}

async function gate(song: Song): Promise<void> {
	const needed = Math.max(0, store.peers.value - 1);
	if (needed <= 0) return;

	if (gateState) clearTimeout(gateState.timer);

	let resolve = () => {};
	const state = {
		songId: song.id,
		needed,
		count: 0,
		resolve: () => resolve(),
		timer: 0,
	};
	const waiting = new Promise<void>((r) => {
		resolve = r;
	});
	state.timer = window.setTimeout(() => {
		if (gateState === state) {
			gateState = null;
			state.resolve();
		}
	}, 4000);
	gateState = state;

	transport.send({ t: "prepare", songId: song.id, youtubeId: song.youtubeId });
	await Promise.all([waiting, warm(song, 4000)]);
}

function markReady(songId: string): void {
	if (!gateState || gateState.songId !== songId) return;
	gateState.count++;
	if (gateState.count < gateState.needed) return;
	clearTimeout(gateState.timer);
	const state = gateState;
	gateState = null;
	state.resolve();
}

async function warmRemote(
	m: Extract<PeerMessage, { t: "prepare" }>,
): Promise<void> {
	if (store.role.value !== "guest") return;
	if (m.youtubeId && m.songId) {
		try {
			await SetYouTubeId(m.songId, m.youtubeId);
		} catch {
			// keep going
		}
	}
	const song = queue.value.find((s) => s.id === m.songId);
	if (song) await warm(song);
	transport.send({ t: "ready", songId: m.songId });
}

function applyControl(m: Extract<PeerMessage, { t: "control" }>): void {
	switch (m.action) {
		case "toggle":
			void togglePlayPause();
			break;
		case "seek":
			seekFromLocalControl(m.positionMs / 1000);
			break;
		case "next":
			void nextSong();
			break;
		case "prev":
			void prevSong();
			break;
		case "shuffle":
			toggleShuffle();
			break;
		case "repeat":
			cycleRepeat();
			break;
		case "play":
			void playAt(m.index);
			break;
		case "enqueue":
			enqueue(m.song);
			break;
		case "playSelection": {
			const song = m.songs.find((s) => s.id === m.songId);
			if (song) void playFromQueueSelection(m.songs, song);
			break;
		}
		case "remove":
			void unqueue(m.index);
			break;
		case "move":
			void moveQueue(m.from, m.to);
			break;
		case "moveAfter":
			void moveAfterCurrent(m.index);
			break;
	}
}

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
	setPlaybackRate(1);
	applyColor(m.color, m.binary);
	if (m.at < controlAt) return;
	if (m.shuffle !== undefined) setShuffle(m.shuffle);
	if (m.repeat !== undefined) setRepeat(m.repeat);
	if (m.youtubeId && m.songId) {
		try {
			await SetYouTubeId(m.songId, m.youtubeId);
		} catch {
			// keep going
		}
	}

	const sameSong = currentSong.value?.id === m.songId;

	currentIndex.value = m.index;
	preloadUpcomingSongs(queue.value, m.index);

	if (!sameSong) {
		const song = queue.value[m.index];
		if (!song) return;
		if (m.playing) {
			await play(song, () => positionAt(m));
		} else {
			await prepareSong(song, () => positionAt(m));
		}
		return;
	}

	if (m.playing && !isPlaying.value) {
		await togglePlayPause();
	} else if (!m.playing && isPlaying.value) {
		await togglePlayPause();
	}
	seek(positionAt(m));
}

function applyHeartbeat(m: Extract<PeerMessage, { t: "heartbeat" }>): void {
	if (currentSong.value?.id !== m.songId) return;
	applyColor(m.color, m.binary);
	if (m.at < controlAt || isLoading.value) return;

	if (m.playing !== isPlaying.value) {
		setPlaybackRate(1);
		void togglePlayPause();
		return;
	}
	if (!m.playing) return;

	const target = projected(m.positionMs, m.at);
	const drift = getPlaybackSeconds() - target;
	if (Math.abs(drift) > 1.5) {
		setPlaybackRate(1);
		seek(target);
	} else if (Math.abs(drift) > 0.08) {
		setPlaybackRate(drift > 0 ? 0.97 : 1.03);
	} else {
		setPlaybackRate(1);
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

function positionAt(m: {
	positionMs: number;
	at: number;
	playing: boolean;
}): number {
	return m.playing ? projected(m.positionMs, m.at) : m.positionMs / 1000;
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

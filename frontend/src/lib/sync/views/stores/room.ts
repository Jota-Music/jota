import { SetYouTubeId } from "@bindings/app";
import { effect } from "@preact/signals";
import type { ControlAction, Song } from "@/lib/music/model";
import {
	currentSong,
	dragSeeking,
	getPlaybackSeconds,
	isLoading,
	isPlaying,
	pause,
	pendingStart,
	prepareSong,
	resume,
	seek,
	seekTo,
	stopPlayer,
	togglePlayPause,
} from "@/lib/music/views/stores/audio";
import { preloadUpcomingSongs } from "@/lib/music/views/stores/player";
import {
	applyShuffle,
	currentIndex,
	cycleRepeat,
	persistQueue,
	queue,
	repeat,
	setRepeat,
	shuffle,
} from "@/lib/music/views/stores/queue";
import { control, playGate } from "@/lib/music/views/stores/remote";
import { binaryColor, dominantColor } from "@/lib/music/views/stores/theme";
import { translateError } from "@/lib/shared/i18n/errors";
import { addError } from "@/lib/shared/views/stores/errors";
import * as transport from "@/lib/sync/app/transport";
import type { Playback, ServerMessage } from "@/lib/sync/model";
import * as store from "@/lib/sync/views/stores";
import * as clock from "@/lib/sync/views/stores/clock";

// The load round: announce a track, every member loads it paused, answers
// `ready`, and the relay releases a shared `play` instant. Both the member that
// pressed the track and the ones receiving the prepare run the same path.
let consensusGen: string | null = null;
// Bumped whenever a round starts, so a join snapshot can tell it was superseded
// by a track change and must not override the track the room is switching to.
let roundSeq = 0;
let playWait: {
	gen: string;
	resolve: (play: { at: number }) => void;
	timer: number;
} | null = null;

// Host session id (from the relay), so a member can drop frames from a previous
// session. The queue is broadcast on every local edit; `joined` gates that until
// the room has injected its own queue, and `suppressQueue` swallows the echo of
// a queue applied from the room.
let roomEpoch = "";
let joined = false;
let suppressQueue = false;
let joinAt = 0;

// The seat holder publishes its playback as the drift reference. The seat is
// inherited, so the sender follows whoever holds it at the time. A peer pulls
// back onto it, but only once its own last control has settled, so the passive
// heartbeat never fights a fresh control or a round.
const SYNC_PERIOD = 5000;
const SYNC_DRIFT = 0.3;
const CONTROL_SETTLE = 3000;
let lastSync = 0;
let lastControl = 0;

function live(): boolean {
	return store.status.value === "open" && store.role.value !== "off";
}

function requestJoin(): void {
	joinAt = Date.now();
	transport.send({ t: "join", at: joinAt });
}

function currentPlayback(): Playback {
	return {
		playing: isPlaying.value,
		positionMs: Math.round(getPlaybackSeconds() * 1000),
		songId: currentSong.value?.id ?? "",
		song: currentSong.value ?? undefined,
		youtubeId: currentSong.value?.youtubeId,
		index: currentIndex.value,
		shuffle: shuffle.value,
		repeat: repeat.value,
		color: dominantColor.value,
		binary: binaryColor.value,
	};
}

// The host answers a join with its live playback; the relay stamps the clock.
function answerJoin(from: string): void {
	if (store.role.value !== "host" || !live()) return;
	transport.send({ t: "snapshot", to: from, state: currentPlayback() });
}

function cancelWait(): void {
	if (!playWait) return;
	clearTimeout(playWait.timer);
	const wait = playWait;
	playWait = null;
	wait.resolve({ at: clock.serverNow() });
}

function waitForPlay(gen: string): Promise<{ at: number }> {
	return new Promise<{ at: number }>((resolve) => {
		const wait = {
			gen,
			resolve,
			timer: window.setTimeout(() => {
				if (playWait !== wait) return;
				playWait = null;
				pendingStart.value = false;
				addError(new Error("consensus timeout"), "room");
				resolve({ at: clock.serverNow() });
			}, 20000),
		};
		playWait = wait;
	});
}

function resolvePlay(gen: string, at: number): void {
	if (!playWait || playWait.gen !== gen) return;
	clearTimeout(playWait.timer);
	const wait = playWait;
	playWait = null;
	wait.resolve({ at });
}

async function settle(gen: string, song: Song): Promise<void> {
	// A newer round supersedes the one in flight: release its wait so it stops
	// owning the spinner and playback.
	roundSeq++;
	cancelWait();
	consensusGen = gen;
	pendingStart.value = true;

	if (song.youtubeId) {
		try {
			await SetYouTubeId(song.id, song.youtubeId);
		} catch {
			// keep going
		}
	}
	const loaded = await prepareSong(song);
	if (consensusGen !== gen) return;
	// A failed load must not stall the room: the relay releases on any answer.
	transport.send({ t: "ready", gen, ok: loaded });

	const play = await waitForPlay(gen);
	if (consensusGen !== gen) return;
	if (!loaded) {
		pendingStart.value = false;
		return;
	}
	await clock.sleep(clock.delayUntil(play.at));
	if (consensusGen !== gen) return;
	await resume();
	pendingStart.value = false;
}

// Anyone may announce the next track: the presser (or the member that reached
// the end of the current one) starts the round, and every other member follows
// the prepare through the same settle path.
async function start(song: Song): Promise<void> {
	if (!live()) return;
	const gen = crypto.randomUUID();
	consensusGen = gen;
	pendingStart.value = true;
	transport.send({
		t: "prepare",
		gen,
		epoch: roomEpoch,
		songId: song.id,
		song,
		youtubeId: song.youtubeId,
		index: queue.value.findIndex((s) => s.id === song.id),
		shuffle: shuffle.value,
		repeat: repeat.value,
		color: dominantColor.value,
		binary: binaryColor.value,
	});
	await settle(gen, song);
}

async function handlePrepare(
	m: Extract<ServerMessage, { t: "prepare" }>,
): Promise<void> {
	if (!live()) return;
	currentIndex.value = m.index;
	if (m.shuffle !== undefined) applyShuffle(m.shuffle);
	if (m.repeat !== undefined) setRepeat(m.repeat);
	applyColor(m.color, m.binary);
	preloadUpcomingSongs(queue.value, m.index);
	if (!m.song) return;
	await settle(m.gen, m.song);
}

// The join reply: its four timestamps fix the clock offset, then the queue and
// the live playback are injected into the local player.
async function applySnapshot(
	m: Extract<ServerMessage, { t: "snapshot" }>,
): Promise<void> {
	clock.sync(joinAt, m.echo, m.at);
	joined = true;
	store.joined.value = true;
	const seq = roundSeq;

	if (m.queue) {
		applyQueue(m.queue);
	} else if (store.role.value === "host") {
		// A fresh room inherits the host queue.
		broadcastQueue();
	}

	const p = m.state;
	if (!p || (p.songId == null && !p.song)) {
		// The room holds a queue but no playback: adopt the queue and stop the
		// local track instead of keeping the one we were playing outside.
		if (m.queue) stopPlayer();
		pendingStart.value = false;
		return;
	}
	applyColor(p.color, p.binary);
	if (p.shuffle !== undefined) applyShuffle(p.shuffle);
	if (p.repeat !== undefined) setRepeat(p.repeat);
	if (p.youtubeId && p.songId) {
		try {
			await SetYouTubeId(p.songId, p.youtubeId);
		} catch {
			// keep going
		}
	}

	const song = p.song ?? queue.value.find((s) => s.id === p.songId) ?? null;
	if (!song) {
		pendingStart.value = false;
		return;
	}

	// A round owns playback once it starts: a join snapshot must not override
	// the track the room is switching to, or abort the round mid-load. The
	// round's own settle plays the new track.
	if (pendingStart.value || roundSeq !== seq) return;

	pendingStart.value = true;
	currentIndex.value = p.index ?? 0;
	preloadUpcomingSongs(queue.value, currentIndex.value);

	const loaded = await prepareSong(song);
	if (roundSeq !== seq) return;
	if (!loaded) {
		pendingStart.value = false;
		return;
	}
	// Align to the room before starting. The target is projected over the relay
	// clock on every sample, so the load and seek time stay inside the margin:
	// the first seek is the jump to the live position, the next one lands close
	// and is fast. A released round carries a future `at` and a zero position, so
	// the seek is a no-op and the shared instant is honored by sleeping to it.
	const playing = p.playing === true;
	const at = p.at ?? m.at;
	const target = () => clock.projected(p.positionMs ?? 0, at, playing);
	for (let i = 0; i < 3; i++) {
		await seekTo(target());
		if (!playing) break;
		if (Math.abs(target() - getPlaybackSeconds()) < 0.15) break;
	}
	if (playing) {
		const delay = clock.delayUntil(at);
		if (delay > 0) await clock.sleep(delay);
		await resume();
		// Absorb the playback start latency, which the target sample predates.
		const drift = target() - getPlaybackSeconds();
		if (Math.abs(drift) > 0.15) seek(target());
	}
	pendingStart.value = false;
}

function applyQueue(input: string | Song[]): void {
	let parsed: unknown;
	try {
		parsed = typeof input === "string" ? JSON.parse(input) : input;
	} catch {
		console.error("sync: invalid queue");
		return;
	}
	if (!Array.isArray(parsed)) return;
	suppressQueue = true;
	queue.value = parsed as Song[];
	persistQueue();
}

function broadcastQueue(): void {
	if (!live()) return;
	transport.send({ t: "queue", data: JSON.stringify(queue.value) });
}

function applyControl(a: ControlAction): void {
	lastControl = Date.now();
	switch (a.action) {
		case "toggle":
			void togglePlayPause();
			break;
		case "play":
			void resume();
			break;
		case "pause":
			pause();
			break;
		case "seek":
			seek(a.positionMs / 1000);
			break;
		case "shuffle":
			applyShuffle(a.on ?? !shuffle.value, a.seed);
			break;
		case "repeat":
			cycleRepeat();
			break;
	}
}

// Controls are local: the caller applies them, and this broadcasts the event so
// the other members apply the same one.
function publish(a: ControlAction): void {
	if (!live()) return;
	lastControl = Date.now();
	transport.send({ t: "control", ...a });
}

// The seat holder's heartbeat: every other member pulls back onto it. A peer
// stuck on another track rejoins outright; one that drifted seeks onto the
// reference; one that missed a play/pause follows it too. A control applied
// within the settle window is left alone, so a fresh local change is never
// fought, and a load or seek in flight is skipped.
function applySync(m: Extract<ServerMessage, { t: "sync" }>): void {
	if (!live() || !joined) return;
	if (pendingStart.value || isLoading.value || dragSeeking.value) return;
	if (Date.now() - lastControl < CONTROL_SETTLE) return;
	if ((currentSong.value?.id ?? "") !== (m.songId ?? "")) {
		if (Date.now() - joinAt > CONTROL_SETTLE) requestJoin();
		return;
	}
	if (m.playing !== isPlaying.value) {
		if (m.playing) void resume();
		else pause();
	}
	const target = clock.projected(
		m.positionMs,
		m.at ?? clock.serverNow(),
		m.playing,
	);
	if (Math.abs(target - getPlaybackSeconds()) > SYNC_DRIFT) seek(target);
}

function handleMessage(msg: ServerMessage): void {
	switch (msg.t) {
		case "role":
			store.role.value = msg.role;
			return;
		case "error":
			store.error.value = translateError(msg.reason);
			addError(msg.reason, "room");
			transport.abandon();
			return;
		case "pong":
			clock.applyPong(msg);
			return;
		case "members":
			store.peers.value = msg.count;
			if (msg.epoch && msg.epoch !== roomEpoch) roomEpoch = msg.epoch;
			return;
		case "join":
			answerJoin(msg.from);
			return;
		case "snapshot":
			void applySnapshot(msg);
			return;
		case "play":
			resolvePlay(msg.gen, msg.at);
			return;
	}
	if (!live()) return;
	switch (msg.t) {
		case "prepare":
			void handlePrepare(msg);
			break;
		case "sync":
			applySync(msg);
			break;
		case "control":
			applyControl(msg);
			break;
		case "queue":
			applyQueue(msg.data);
			break;
	}
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

transport.onMessage(handleMessage);

transport.onOpen(() => {
	joined = false;
	store.joined.value = false;
	requestJoin();
});

transport.onClosed(() => {
	cancelWait();
	consensusGen = null;
	pendingStart.value = false;
	roomEpoch = "";
	joined = false;
	suppressQueue = false;
	store.joined.value = false;
});

// The room surface is live whenever we are connected and seated.
effect(() => {
	const on = live();
	control.value = on ? publish : null;
	playGate.value = on ? start : null;
});

// Any local queue edit is broadcast to the room; a queue applied from the room is
// not echoed back.
effect(() => {
	const q = queue.value;
	if (suppressQueue) {
		suppressQueue = false;
		return;
	}
	if (!joined || !live()) return;
	transport.send({ t: "queue", data: JSON.stringify(q) });
});

// Every member feeds the relay cache, so the room snapshot tracks the pause,
// seek and track changes even while the host is away. The live host answer is
// still the primary join path; the cache is the fallback. The seat holder also
// publishes a slower heartbeat so peers can pull back onto its position.
setInterval(() => {
	if (!live()) return;
	transport.send({ t: "state", ...currentPlayback() });
	const now = Date.now();
	if (store.role.value !== "host" || pendingStart.value) return;
	if (now - lastSync < SYNC_PERIOD) return;
	lastSync = now;
	transport.send({ t: "sync", ...currentPlayback(), at: clock.serverNow() });
}, 2000);

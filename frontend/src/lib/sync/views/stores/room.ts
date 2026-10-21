import { SetYouTubeId } from "@bindings/app";
import { effect } from "@preact/signals";
import { delta, insertAt } from "@/lib/music/app/queue";
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
} from "@/lib/music/views/stores/audio";
import { preloadUpcomingSongs } from "@/lib/music/views/stores/player";
import {
	applyShuffle,
	currentIndex,
	nextRepeat,
	persistQueue,
	queue,
	repeat,
	replaceQueue,
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
	resolve: (play: Release) => void;
	timer: number;
} | null = null;

type Release = { at: number; positionMs: number };

// Host session id (from the relay), so a member can drop frames from a previous
// session. Local queue edits are diffed against `lastQueue`: an incremental one
// rides a control op, anything else is a full snapshot. `joined` gates that until
// the room has injected its own queue, and `suppressedQueueRef` swallows the echo
// of a queue applied from the room.
let roomEpoch = "";
let joined = false;
let suppressedQueueRef: Song[] | null = null;
let lastQueue: Song[] | null = null;
let memberCount = 0;
let joinTimer: number | null = null;
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

function clearJoin(): void {
	if (joinTimer == null) return;
	clearTimeout(joinTimer);
	joinTimer = null;
}

// A join the relay never answers would leave `joined` false forever, silently
// swallowing every queue edit, so a silent snapshot is asked for again.
function requestJoin(): void {
	joinAt = Date.now();
	transport.send({ t: "join", at: joinAt });
	clearJoin();
	joinTimer = window.setTimeout(() => {
		joinTimer = null;
		if (joined) return;
		requestJoin();
	}, 4000);
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

// An empty room has no host, so the relay seats us as host. Our local session
// then becomes the room's: broadcast our queue and immediate now-playing
// instead of adopting a stale relay cache from a previous session. A room that
// already holds playback is not empty: adopting it is how a host lands on the
// track the room is playing instead of freezing everyone on its own copy.
function inherit(state?: Playback): boolean {
	if (store.role.value !== "host" || !currentSong.value) return false;
	if (state && (state.songId || state.song)) return false;
	transport.send({ t: "state", ...currentPlayback() });
	pendingStart.value = false;
	return true;
}

function cancelWait(): void {
	if (!playWait) return;
	clearTimeout(playWait.timer);
	const wait = playWait;
	playWait = null;
	wait.resolve({ at: clock.serverNow(), positionMs: 0 });
}

function waitForPlay(gen: string): Promise<Release> {
	return new Promise<Release>((resolve) => {
		const wait = {
			gen,
			resolve,
			timer: window.setTimeout(() => {
				if (playWait !== wait || consensusGen !== gen) return;
				playWait = null;
				pendingStart.value = false;
				// Abandon the round so settle returns without starting solo, and
				// pull the room snapshot to land on the real track and position.
				consensusGen = null;
				addError(new Error("consensus timeout"), "room");
				requestJoin();
				resolve({ at: clock.serverNow(), positionMs: 0 });
			}, 8000),
		};
		playWait = wait;
	});
}

function resolvePlay(
	gen: string,
	play: Extract<ServerMessage, { t: "play" }>,
): void {
	if (!playWait || playWait.gen !== gen) return;
	clearTimeout(playWait.timer);
	const wait = playWait;
	playWait = null;
	wait.resolve({ at: play.at, positionMs: play.positionMs ?? 0 });
}

async function settle(gen: string, song: Song): Promise<void> {
	// A newer round supersedes the one in flight: release its wait so it stops
	// owning the spinner and playback.
	roundSeq++;
	cancelWait();
	consensusGen = gen;
	pendingStart.value = true;

	// Registered before the load: the relay releases the round on its deadline,
	// so the play frame can land while this member is still loading the track.
	const play = waitForPlay(gen);

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

	const release = await play;
	if (consensusGen !== gen) return;
	if (!loaded) {
		pendingStart.value = false;
		return;
	}
	await clock.sleep(clock.delayUntil(release.at));
	if (consensusGen !== gen) return;
	// A release whose instant already passed (this member was still loading when
	// the round expired) lands on the live position instead of from the top.
	await seekTo(clock.projected(release.positionMs, release.at, true));
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
	clearJoin();
	store.joined.value = true;

	// The host owns the queue whether or not it has anything loaded: adopting
	// the relay cache instead is how a stale queue outlives the session that
	// wrote it. A guest takes the room's queue, which is the only queue there is.
	if (store.role.value === "host") {
		broadcastQueue();
	} else if (m.queue) {
		applyQueue(m.queue);
	}

	if (inherit(m.state)) return;
	const seq = roundSeq;

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
	const songs = parsed as Song[];
	lastQueue = songs;
	// Re-broadcasts and the relay cache echo the queue we already hold; swapping
	// it for an identical array would only re-render the whole panel.
	if (JSON.stringify(songs) === JSON.stringify(queue.value)) return;
	suppressedQueueRef = songs;
	replaceQueue(songs, currentSong.value?.id);
}

function broadcastQueue(): void {
	if (!live()) return;
	lastQueue = queue.value;
	transport.send({ t: "queue", data: JSON.stringify(queue.value) });
}

function applyControl(a: ControlAction): void {
	lastControl = Date.now();
	switch (a.action) {
		case "play":
			void resume();
			break;
		case "pause":
			pause();
			break;
		case "stop":
			stopPlayer();
			break;
		case "seek":
			seek(a.positionMs / 1000);
			break;
		case "shuffle":
			applyShuffle(a.on ?? !shuffle.value, a.seed);
			break;
		case "repeat":
			setRepeat(a.mode ?? nextRepeat(repeat.value));
			break;
		case "enqueue": {
			// A remote add lands where the sender put it, without replacing (or
			// re-rendering) the rest of the queue. The position rides the message,
			// so a repeated id cannot send it to a different row here.
			const next = insertAt(queue.value, a.songs, a.at);
			suppressedQueueRef = next;
			lastQueue = next;
			replaceQueue(next, currentSong.value?.id);
			break;
		}
		case "youtube": {
			const items = queue.value;
			const idx = items.findIndex((song) => song.id === a.id);
			if (idx === -1 || items[idx].youtubeId === a.youtubeId) break;
			const next = items.slice();
			next[idx] = { ...next[idx], youtubeId: a.youtubeId };
			suppressedQueueRef = next;
			lastQueue = next;
			queue.value = next;
			persistQueue();
			break;
		}
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
			// Queue ops do not feed the relay cache, so a newcomer's join snapshot
			// could miss recent adds: the host refreshes the cached queue whenever
			// the membership grows.
			if (msg.count > memberCount && joined && store.role.value === "host") {
				broadcastQueue();
			}
			memberCount = msg.count;
			return;
		case "join":
			answerJoin(msg.from);
			return;
		case "snapshot":
			void applySnapshot(msg);
			return;
		case "play":
			resolvePlay(msg.gen, msg);
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
	if (color) dominantColor.value = color;
	if (binary) binaryColor.value = binary;
}

transport.onMessage(handleMessage);

transport.onOpen(() => {
	joined = false;
	store.joined.value = false;
	requestJoin();
});

transport.onClosed(() => {
	cancelWait();
	clearJoin();
	consensusGen = null;
	pendingStart.value = false;
	roomEpoch = "";
	joined = false;
	suppressedQueueRef = null;
	lastQueue = null;
	memberCount = 0;
	store.joined.value = false;
});

// The room surface is live whenever we are connected and seated.
effect(() => {
	const on = live();
	control.value = on ? publish : null;
	playGate.value = on ? start : null;
});

// Any local queue edit reaches the room; a queue applied from the room is not
// echoed back.
effect(() => {
	const q = queue.value;
	if (suppressedQueueRef === q) {
		suppressedQueueRef = null;
		lastQueue = q;
		return;
	}
	if (!joined || !live()) return;
	// An incremental edit (an added song, a resolved id) rides a control op, so
	// adding one track no longer rewrites every member's queue. Anything
	// structural is still sent whole, which also refreshes the relay's join cache.
	const op = lastQueue ? delta(lastQueue, q) : null;
	lastQueue = q;
	if (op) {
		transport.send({ t: "control", ...op });
		return;
	}
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

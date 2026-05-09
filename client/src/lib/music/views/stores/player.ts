import type { Song } from "@/lib/music/model";
import {
    audioDuration,
    currentSong,
    getPlaybackSeconds,
    prepareSong,
    seek,
    setOnTrackEnded,
    socketSeek,
    stopPlayer,
    syncPlayerFromServer,
    togglePlayPause,
} from "@/lib/music/views/stores/audio";
import { AudioCache } from "@/lib/music/views/stores/cache";
import {
    broadcastSeek,
    currentIndex,
    queue,
    shareSnapshot,
} from "@/lib/music/views/stores/queue";
import { ws } from "@/lib/shared/api/socket";

const SAME_TRACK_SEEK_EPS_S = 0.4;

let remotePlaybackTail: Promise<void> = Promise.resolve();

function enqueueRemotePlayback(fn: () => void | Promise<void>): Promise<void> {
    const p = remotePlaybackTail.catch(() => {}).then(fn);
    remotePlaybackTail = p;
    return p;
}

/**
 * Internal token used to ensure only the latest async playback
 * operation is allowed to complete (prevents race conditions).
 */
let currentPlayerToken: string | null = null;

/**
 * Sets the current player token.
 *
 * @param id Unique identifier for the current playback operation.
 */
function setPlayerToken(id: string | null) {
    currentPlayerToken = id;
}

/**
 * Consumes the current token if it matches the provided one.
 *
 * @param syncId Token to validate.
 * @returns True if the token matches and was consumed.
 */
function consumePlayerToken(syncId: string): boolean {
    if (currentPlayerToken !== syncId) return false;
    currentPlayerToken = null;
    return true;
}

/**
 * Synchronizes the player with the server only if the given token
 * is still valid (i.e., no newer playback has started).
 *
 * @param syncId Token of the playback operation.
 */
function syncIfCurrentPlayer(syncId: string) {
    if (consumePlayerToken(syncId)) {
        syncPlayerFromServer(true);
    }
}

/**
 * Preloads the next 1–2 songs to improve playback continuity.
 *
 * @param songs Full queue.
 * @param idx Current index.
 */
function preloadUpcomingSongs(songs: Song[], idx: number) {
    const upcoming = songs.slice(idx + 1, idx + 3);
    if (upcoming.length > 0) void AudioCache.preload(...upcoming);
}

function clampPlaybackSeconds(seconds: number): number {
    if (!Number.isFinite(seconds) || seconds < 0) return 0;
    const d = audioDuration.value;
    if (Number.isFinite(d) && d > 0) {
        const eps = 0.05;
        return Math.min(seconds, Math.max(0, d - eps));
    }
    return seconds;
}

/** Seek desde la UI local: mueve el audio y notifica a los peers (sin timers). */
export function seekFromLocalControl(seconds: number) {
    const t = clampPlaybackSeconds(seconds);
    seek(t);
    broadcastSeek(t);
}

export function applyRemoteSeek(seconds: number) {
    return enqueueRemotePlayback(() => {
        if (!Number.isFinite(seconds) || seconds < 0) return;
        socketSeek(clampPlaybackSeconds(seconds));
    });
}

type ReconcileSource = "local" | "remote";

async function reconcileQueueAndPlayback(
    nextQueue: Song[],
    nextIndex: number,
    source: ReconcileSource,
    remotePlaying: boolean | undefined,
    remotePosition?: number,
) {
    const prevId = currentSong.value?.id ?? null;

    queue.value = nextQueue;
    currentIndex.value = nextIndex;

    const song =
        nextIndex >= 0 && nextIndex < nextQueue.length
            ? nextQueue[nextIndex]
            : null;

    if (!song) {
        stopPlayer();
        return;
    }

    const sameTrack = prevId === song.id;

    if (sameTrack) {
        // Siempre alinear con el objeto del snapshot/cola: en remoto el broadcast
        // puede traer metadata distinta (p. ej. covers) con el mismo id; si no
        // actualizamos, currentSong queda obsoleto y el tema de color no se recalcula.
        currentSong.value = song;

        preloadUpcomingSongs(nextQueue, nextIndex);
        if (source === "remote") {
            if (
                remotePosition !== undefined &&
                Number.isFinite(remotePosition) &&
                Math.abs(remotePosition - getPlaybackSeconds()) > SAME_TRACK_SEEK_EPS_S
            ) {
                socketSeek(clampPlaybackSeconds(remotePosition));
            }
            if (remotePlaying !== undefined) {
                syncPlayerFromServer(remotePlaying);
            }
        }
        return;
    }

    const syncId = newPlayerId();
    setPlayerToken(syncId);

    if (source === "local") {
        shareSnapshot({ position: 0, playing: true });
    }

    const startSeconds =
        source === "remote" &&
        remotePosition !== undefined &&
        Number.isFinite(remotePosition)
            ? Math.max(0, remotePosition)
            : undefined;

    await prepareSong(song, startSeconds);

    if (source === "local") {
        syncIfCurrentPlayer(syncId);
    } else if (remotePlaying !== undefined && consumePlayerToken(syncId)) {
        syncPlayerFromServer(remotePlaying);
    }
}

async function applyQueueState(nextQueue: Song[], nextIndex: number) {
    await reconcileQueueAndPlayback(nextQueue, nextIndex, "local", undefined);
    shareSnapshot();
}

export function applyRoomPlaybackFromPeer(
    nextQueue: Song[],
    nextIndex: number,
    playing: boolean,
    position?: number,
): Promise<void> {
    return enqueueRemotePlayback(() =>
        reconcileQueueAndPlayback(
            nextQueue.slice(),
            nextIndex,
            "remote",
            playing,
            position,
        ),
    );
}

function newPlayerId() {
    return crypto.randomUUID();
}

export async function playFromQueueSelection(
    fullOrderedSongs: Song[],
    clicked: Song,
) {
    const i = fullOrderedSongs.findIndex((s) => s.id === clicked.id);
    if (i === -1) return;

    await applyQueueState(fullOrderedSongs.slice(), i);
}

/**
 * Inserts a song right after the currently playing one.
 * If the queue is empty, initializes it without starting playback.
 *
 * @param song Song to enqueue.
 */
export function enqueue(song: Song) {
    const newQueue = [...queue.value];
    const newIndex = currentIndex.value;

    if (newQueue.length === 0) {
        newQueue.push(song);
        queue.value = newQueue;
        currentIndex.value = 0;
        shareSnapshot();
        void AudioCache.preload(song);
        return;
    }

    const insertAt =
        newIndex >= 0 && newIndex < newQueue.length ? newIndex + 1 : newQueue.length;

    newQueue.splice(insertAt, 0, song);
    queue.value = newQueue;
    shareSnapshot();

    void AudioCache.preload(song);
}

/**
 * Removes a song from the queue and keeps playback consistent.
 *
 * @param removeIdx Index to remove.
 */
export async function unqueue(removeIdx: number): Promise<void> {
    const q = [...queue.value];
    const n = q.length;
    if (removeIdx < 0 || removeIdx >= n) return;

    const playingId =
        currentIndex.value >= 0 && currentIndex.value < n
            ? q[currentIndex.value].id
            : null;

    q.splice(removeIdx, 1);

    let nextIndex: number;

    if (q.length === 0) {
        nextIndex = -1;
    } else if (playingId != null) {
        const j = q.findIndex((s) => s.id === playingId);
        nextIndex = j !== -1 ? j : Math.min(removeIdx, q.length - 1);
    } else {
        nextIndex = 0;
    }

    await applyQueueState(q, nextIndex);
}

/**
 * Moves a song within the queue.
 *
 * @param from Source index.
 * @param to Target index.
 */
export async function moveQueue(from: number, to: number): Promise<void> {
    const a = [...queue.value];
    const n = a.length;
    if (from < 0 || from >= n) return;

    const clamped = Math.max(0, Math.min(n - 1, to));
    if (from === clamped) return;

    const playingId =
        currentIndex.value >= 0 && currentIndex.value < n
            ? a[currentIndex.value].id
            : null;

    const [it] = a.splice(from, 1);
    a.splice(clamped, 0, it);

    const nextIndex =
        playingId != null
            ? a.findIndex((s) => s.id === playingId)
            : a.length > 0
                ? 0
                : -1;

    await applyQueueState(
        a,
        nextIndex >= 0 ? nextIndex : a.length > 0 ? 0 : -1,
    );
}

/**
 * Moves a song to be played immediately after the current one.
 *
 * @param from Index of the song to move.
 */
export async function moveAfterCurrent(from: number): Promise<void> {
    const a = [...queue.value];
    const n = a.length;
    const ci = currentIndex.value;

    if (from < 0 || from >= n) return;
    if (ci < 0 || ci >= n) return;
    if (from === ci || from === ci + 1) return;

    const playingId = a[ci].id;

    const [it] = a.splice(from, 1);

    const ci2 = a.findIndex((s) => s.id === playingId);
    if (ci2 === -1) return;

    const insertAt = Math.min(ci2 + 1, a.length);
    a.splice(insertAt, 0, it);

    const nextIndex = a.findIndex((s) => s.id === playingId);

    await applyQueueState(a, nextIndex >= 0 ? nextIndex : 0);
}

/**
 * Jumps to a specific index in the queue and starts playback.
 *
 * @param i Target index.
 */
export async function playAt(i: number): Promise<void> {
    const q = queue.value;

    if (i < 0 || i >= q.length) return;
    if (i === currentIndex.value) return;

    const syncId = newPlayerId();
    currentIndex.value = i;
    setPlayerToken(syncId);

    shareSnapshot({ position: 0, playing: true });

    await prepareSong(q[i]);

    syncIfCurrentPlayer(syncId);
    shareSnapshot();
}

/**
 * Toggles play/pause state.
 */
export async function toggleSong() {
    await togglePlayPause();
    ws.send("toggle");
}

/**
 * Advances to the next song in the queue.
 */
export async function nextSong() {
    const q = queue.value;
    const i = currentIndex.value;

    if (i >= q.length - 1) return;

    await playAt(i + 1);
}

/**
 * Goes back to the previous song in the queue.
 */
export async function prevSong() {
    const i = currentIndex.value;

    if (i <= 0) return;

    await playAt(i - 1);
}

/**
 * Handles automatic progression when a track ends.
 */
async function handleTrackEnd() {
    const q = queue.value;
    const i = currentIndex.value;

    if (i >= q.length - 1) return;

    await playAt(i + 1);
}

/**
 * Registers autoplay behavior when the current track finishes.
 */
setOnTrackEnded(() => {
    void handleTrackEnd();
});
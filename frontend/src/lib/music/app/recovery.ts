// Playback recovery decisions for the shared audio element. A stream that died
// mid-track (network drop) must re-resolve and resume at the same position, so
// the store needs to tell a dead stream apart from ordinary buffering and space
// its retries out.

// How long the element may stop advancing mid-track before the stream counts as
// dead. A real end is decided by shouldEnd before this ever runs.
export const STALL_MS = 4000;

const BASE_RETRY_MS = 1000;
const MAX_RETRY_MS = 15000;

export function stalled(stillMs: number): boolean {
	return stillMs >= STALL_MS;
}

// Exponential backoff between recovery attempts, capped so a long outage keeps a
// steady retry cadence instead of hammering the resolver.
export function retryDelay(attempt: number): number {
	return Math.min(BASE_RETRY_MS * 2 ** attempt, MAX_RETRY_MS);
}

// How long a stream may accept play() without producing sound before it counts
// as dead. On Android an element paused for a while (screen off, app in the
// background) keeps the URL it resolved, and that URL has since expired: play()
// resolves, no `error` ever fires and nothing reaches the speakers, so the
// session stays paused with nothing scheduled to retry it.
export const START_MS = 6000;

// A resume the user just asked for gets a much shorter window: the stream is
// usually alive, and waiting the full one would make every pause feel slow
// before the recovery it rarely needs.
export const RESUME_MS = 1800;

// How much audio has to come out before the stream counts as playing. A dying
// stream hands over the frames it still had buffered -- about a tenth of a
// second -- and then stops, so anything that short is not sound.
const MOVED_S = 1;

// The slice of HTMLMediaElement a wait touches, so it can be exercised without a
// DOM (the real element is structurally compatible).
type Surface = {
	currentTime: number;
	addEventListener(name: string, listener: () => void): void;
	removeEventListener(name: string, listener: () => void): void;
};

type Wait = {
	el: Surface;
	from: number;
	wall: number;
	until: number;
	done: (started: boolean) => void;
};

// A wait cannot schedule itself. A timer created while the page is hidden never
// fires -- measured on a phone left paused: a 6 s timeout and a 500 ms poll both
// stayed silent for twenty-five minutes, while an interval created before the app
// left the foreground kept ticking on schedule. The store therefore starts one
// ticker at load and every wait rides it.
const waits = new Set<Wait>();

function playingNow(w: Wait): boolean {
	const moved = w.el.currentTime - w.from;
	const wall = (Date.now() - w.wall) / 1000;
	return moved >= MOVED_S && moved >= wall * 0.5;
}

// Called once a second by the store. Settles every wait that either produced
// sound or ran out of time.
export function tick(): void {
	const now = Date.now();
	for (const w of [...waits]) {
		if (playingNow(w)) w.done(true);
		else if (now >= w.until) w.done(false);
	}
}

// Resolves true once the element proves it is playing, false when it reported an
// error or never got there in time. It has to be the position that proves it: the
// element behind the Android report answered `paused: false` and `readyState: 4`
// with the buffer full and no error, while its clock stood still and no sample
// ever reached the speakers.
export function playing(el: Surface, timeoutMs = START_MS): Promise<boolean> {
	return new Promise((resolve) => {
		const done = (started: boolean) => {
			waits.delete(wait);
			el.removeEventListener("error", onError);
			resolve(started);
		};
		const onError = () => done(false);
		const wait: Wait = {
			el,
			from: el.currentTime,
			wall: Date.now(),
			until: Date.now() + timeoutMs,
			done,
		};
		el.addEventListener("error", onError);
		waits.add(wait);
	});
}

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

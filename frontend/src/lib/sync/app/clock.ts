import type { Stamps } from "@/lib/sync/model";

export function offset(s: Stamps): number {
	return (s.t1 - s.t0 + (s.t2 - s.t3)) / 2;
}

export function roundTrip(s: Stamps): number {
	return Math.max(0, s.t3 - s.t0 - (s.t2 - s.t1));
}

// A relay position projected over the relay clock.
export function project(
	positionMs: number,
	at: number,
	now: number,
	playing: boolean,
): number {
	const t = playing ? (positionMs + (now - at)) / 1000 : positionMs / 1000;
	return Math.max(0, t);
}

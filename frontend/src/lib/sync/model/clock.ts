// Pure clock math for a four-timestamp NTP exchange:
//   t0 client send, t1 relay receive, t2 relay send, t3 client receive.
// `offset` maps the local clock onto the relay clock and `roundTrip` bounds the
// error of the sample (a lower round trip is a better sample).
export interface Stamps {
	t0: number;
	t1: number;
	t2: number;
	t3: number;
}

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

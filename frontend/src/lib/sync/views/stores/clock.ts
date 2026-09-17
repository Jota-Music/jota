import { effect } from "@preact/signals";
import { offset as ntpOffset, project, roundTrip } from "@/lib/sync/app/clock";
import * as transport from "@/lib/sync/app/transport";
import type { ServerMessage } from "@/lib/sync/model";
import { offsetMs, status } from "@/lib/sync/views/stores";

// Clock samples against the relay. The best round trip wins the offset: a low
// RTT bounds the one-way error, and a short window follows network changes.
type Sample = { rtt: number; offset: number };
const samples: Sample[] = [];
let pingId = 0;
const pending = new Map<number, number>();

function add(rtt: number, offset: number): void {
	samples.push({ rtt, offset });
	if (samples.length > 12) samples.shift();
	let best = samples[0];
	for (const s of samples) if (s.rtt < best.rtt) best = s;
	offsetMs.value = Math.round(best.offset);
}

export function reset(): void {
	samples.length = 0;
	pending.clear();
}

export function ping(): void {
	const id = ++pingId;
	const at = Date.now();
	pending.set(id, at);
	transport.send({ t: "ping", id, at });
	for (const [key, sentAt] of pending) {
		if (at - sentAt > 5000) pending.delete(key);
	}
}

export function applyPong(m: Extract<ServerMessage, { t: "pong" }>): void {
	const sent = pending.get(m.id);
	if (sent == null) return;
	pending.delete(m.id);
	const rtt = Date.now() - sent;
	add(rtt, m.echo - sent - rtt / 2);
}

// The join reply carries all four stamps of an NTP exchange: our send (t0), the
// relay receive (echo) and send (at), and our receive (t3). A single sample is
// enough to project the snapshot immediately; ping/pong refines it afterwards.
export function sync(t0: number, echo: number, at: number): void {
	const stamps = { t0, t1: echo, t2: at, t3: Date.now() };
	add(roundTrip(stamps), ntpOffset(stamps));
}

export function serverNow(): number {
	return Date.now() + offsetMs.value;
}

// Milliseconds to wait before the relay instant `relayAt`. Negative means it
// already passed. The clamp keeps a bad clock offset from stalling playback.
export function delayUntil(relayAt: number): number {
	return Math.min(relayAt - offsetMs.value - Date.now(), 3000);
}

export function projected(
	positionMs: number,
	at: number,
	playing: boolean,
): number {
	return project(positionMs, at, serverNow(), playing);
}

export function sleep(ms: number): Promise<void> {
	return ms > 0
		? new Promise((resolve) => setTimeout(resolve, ms))
		: Promise.resolve();
}

// Probe the clock right away, with a quick burst, so a round released shortly
// after connecting still has an offset to schedule its start against.
effect(() => {
	if (status.value !== "open") return;
	reset();
	ping();
	const burst = [window.setTimeout(ping, 150), window.setTimeout(ping, 400)];
	const interval = window.setInterval(ping, 1000);
	return () => {
		for (const t of burst) window.clearTimeout(t);
		window.clearInterval(interval);
	};
});

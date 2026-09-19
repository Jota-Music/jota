// End-of-track decision for the shared audio element. WebKit on macOS does not
// reliably fire `ended` for YouTube CDN streams and can report an unreliable or
// Infinity element duration, so the duration resolved from the stream URL is
// authoritative whenever it is known.
export function shouldEnd(
	time: number,
	elementDuration: number,
	knownDuration: number,
	stillMs: number,
): boolean {
	if (!Number.isFinite(time) || time <= 0) return false;

	if (knownDuration > 0 && time >= knownDuration) return true;

	const fromElement = Number.isFinite(elementDuration) && elementDuration > 0;
	if (fromElement && time >= elementDuration - 0.25) return true;

	// No known duration and no usable element duration: only a stall right at
	// the end can signal it, and even then the gate keeps a mid-track network
	// stall from ending the song.
	const gate = knownDuration > 0 ? knownDuration : elementDuration;
	return Number.isFinite(gate) && time >= gate - 2 && stillMs >= 1000;
}

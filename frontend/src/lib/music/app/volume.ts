// The slider is a 0..1 setting, but the low end needs a real quiet floor: a
// linear bar still leaves the minimum too loud to use as an off switch. From
// QUIET_START up it stays linear (the feel users are used to, so lowering is
// responsive and needs no precision); below it a quadratic taper reaches true
// silence. The input is clamped because the arithmetic can land a hair above 1
// on the way down, and audio.volume is only defined up to 1.
const QUIET_START = 0.3;
const QUIET_LEVEL = 0.2;

export function gain(value: number): number {
	const v = Math.min(1, Math.max(0, value));
	if (v >= QUIET_START) {
		return (
			QUIET_LEVEL + ((v - QUIET_START) / (1 - QUIET_START)) * (1 - QUIET_LEVEL)
		);
	}
	const t = v / QUIET_START;
	return t * t * QUIET_LEVEL;
}

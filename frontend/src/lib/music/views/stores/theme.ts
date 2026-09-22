import { effect, signal } from "@preact/signals";

export const dominantColor = signal<string | null>(null);
export const binaryColor = signal<string | null>(null);

effect(() => {
	const dominant = dominantColor.value;
	const binary = binaryColor.value;
	if (dominant) {
		document.documentElement.style.setProperty("--dominant-color", dominant);
	}
	if (binary) {
		document.documentElement.style.setProperty("--binary-color", binary);
	}
});

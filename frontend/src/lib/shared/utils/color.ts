import { cover } from "@/lib/shared/utils/cover";

/* --------------------------------------------------
   NEUTRAL FILTER
-------------------------------------------------- */
function isNeutral(r: number, g: number, b: number): boolean {
	const max = Math.max(r, g, b);
	const min = Math.min(r, g, b);
	return max - min < 20 || max < 25 || min > 230;
}

/* --------------------------------------------------
   RGB KEY
-------------------------------------------------- */
function rgbToKey(r: number, g: number, b: number): string {
	return `${r >> 4}-${g >> 4}-${b >> 4}`;
}

/* --------------------------------------------------
   AVERAGE (original fallback)
-------------------------------------------------- */
function averageRgbFromImageData(
	data: Uint8ClampedArray,
): [number, number, number] | null {
	let tr = 0;
	let tg = 0;
	let tb = 0;
	let n = 0;

	for (let i = 0; i < data.length; i += 16) {
		const a = data[i + 3];
		if (a < 8) continue;

		tr += data[i];
		tg += data[i + 1];
		tb += data[i + 2];
		n++;
	}

	if (n === 0) return null;

	return [Math.round(tr / n), Math.round(tg / n), Math.round(tb / n)];
}

/* --------------------------------------------------
   DOMINANT COLOR (fixed internals)
-------------------------------------------------- */
type Rgb = [number, number, number];

const dominantCache = new Map<string, Promise<Rgb | null>>();
const DOMINANT_CACHE_MAX = 200;

function cachedDominantColor(url: string): Promise<Rgb | null> {
	const hit = dominantCache.get(url);
	if (hit) {
		// Refresh recency so a hot cover survives eviction.
		dominantCache.delete(url);
		dominantCache.set(url, hit);
		return hit;
	}

	const pending = computeDominantColor(url).catch((err) => {
		// A failed fetch is not worth remembering: drop it so the next call
		// can retry instead of replaying the rejection forever.
		dominantCache.delete(url);
		throw err;
	});

	dominantCache.set(url, pending);
	if (dominantCache.size > DOMINANT_CACHE_MAX) {
		const oldest = dominantCache.keys().next().value;
		if (oldest !== undefined) dominantCache.delete(oldest);
	}

	return pending;
}

export function getDominantColorFromImage(url: string): Promise<Rgb | null> {
	return cachedDominantColor(url);
}

async function computeDominantColor(url: string): Promise<Rgb | null> {
	const img = new Image();
	img.crossOrigin = "anonymous";

	await new Promise<void>((resolve, reject) => {
		img.onload = () => resolve();
		img.onerror = reject;
		img.src = cover(url, 160) ?? url;
	});

	if (!img.naturalWidth || !img.naturalHeight) return null;

	const size = 100;

	const canvas = document.createElement("canvas");
	const ctx = canvas.getContext("2d");
	if (!ctx) return null;

	canvas.width = size;
	canvas.height = size;

	ctx.drawImage(img, 0, 0, size, size);

	let data: ImageData;

	try {
		data = ctx.getImageData(0, 0, size, size);
	} catch (e) {
		console.warn("getImageData failed (CORS):", e);
		return null;
	}

	const pixels = data.data;

	const map = new Map<
		string,
		{
			count: number;
			r: number;
			g: number;
			b: number;
		}
	>();

	for (let i = 0; i < pixels.length; i += 8) {
		const r = pixels[i];
		const g = pixels[i + 1];
		const b = pixels[i + 2];
		const a = pixels[i + 3];

		if (a < 8) continue;
		if (isNeutral(r, g, b)) continue;

		const key = rgbToKey(r, g, b);

		const entry = map.get(key) ?? {
			count: 0,
			r: 0,
			g: 0,
			b: 0,
		};

		entry.count++;
		entry.r += r;
		entry.g += g;
		entry.b += b;

		map.set(key, entry);
	}

	let dominant: { count: number; r: number; g: number; b: number } | null =
		null;

	for (const v of map.values()) {
		if (!dominant || v.count > dominant.count) {
			dominant = v;
		}
	}

	if (dominant && dominant.count > 0) {
		return [
			Math.round(dominant.r / dominant.count),
			Math.round(dominant.g / dominant.count),
			Math.round(dominant.b / dominant.count),
		];
	}

	return averageRgbFromImageData(pixels);
}

/* --------------------------------------------------
   BRIGHTNESS (original)
-------------------------------------------------- */
function getBrightness(r: number, g: number, b: number): number {
	return (r * 299 + g * 587 + b * 114) / 1000;
}

/* --------------------------------------------------
   NORMALIZE COLOR (restored)
-------------------------------------------------- */
export function normalizeColor(
	r: number,
	g: number,
	b: number,
): [number, number, number] {
	const brightness = getBrightness(r, g, b);

	let factor = 1;

	if (brightness < 80) {
		factor = 80 / brightness;
	} else if (brightness > 180) {
		factor = 180 / brightness;
	}

	const clamp = (v: number) =>
		Math.max(0, Math.min(255, Math.round(v * factor)));

	return [clamp(r), clamp(g), clamp(b)];
}

/* --------------------------------------------------
   ICON CONTRAST (restored)
-------------------------------------------------- */
export function isDarkColor(r: number, g: number, b: number): boolean {
	return getBrightness(r, g, b) < 140;
}

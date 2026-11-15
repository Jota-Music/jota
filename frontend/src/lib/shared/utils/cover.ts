import { isAndroid } from "@/lib/shared/utils/platform";

const PROXY_HOSTS = new Set([
	"i.scdn.co",
	"mosaic.scdn.co",
	"misc.scdn.co",
	"lh3.googleusercontent.com",
]);

const YOUTUBE_HOSTS = new Set(["i.ytimg.com", "img.youtube.com"]);

const YOUTUBE_VARIANTS = [
	{ max: 160, variant: "mqdefault" },
	{ max: 360, variant: "hqdefault" },
] as const;

// Spotify keeps the same art at several sizes, and the size lives in the first
// 16 characters of the image id: 8 of namespace, 4 fixed, then the variant.
// Asking the CDN for the variant the UI paints is what keeps a row of 80px
// thumbnails from downloading 640px covers.
const SPOTIFY_VARIANTS: Record<string, { code: string; width: number }[]> = {
	// Albums. Measured: 77 KB at 640, 21 KB at 300, 2 KB at 64.
	ab67616d: [
		{ code: "b273", width: 640 },
		{ code: "1e02", width: 300 },
		{ code: "4851", width: 64 },
	],
	// Artists. Measured: 76 KB at 531, 23 KB at 265, 7.6 KB at 133.
	ab676161: [
		{ code: "e5eb", width: 531 },
		{ code: "5174", width: 265 },
		{ code: "f178", width: 133 },
	],
};

const SPOTIFY_IMAGE = "i.scdn.co";

// The CDN steps straight from 300 to 640, so a slot of 320 would pay four times
// the bytes and a 640px decode for art 6% wider than the variant it gets. Sizes
// that really need the bigger step (384, 480) stay above it.
const SLACK = 0.9;

function hostOf(url: string): string | null {
	try {
		return new URL(url).hostname;
	} catch {
		return null;
	}
}

function youtube(url: string, size: number): string {
	// hqdefault is the largest variant every video has: sddefault, hq720 and
	// maxresdefault answer 404 for anything uploaded without HD art.
	const variant =
		YOUTUBE_VARIANTS.find((entry) => size <= entry.max)?.variant ?? "hqdefault";
	const parsed = new URL(url);
	const slash = parsed.pathname.lastIndexOf("/");
	if (slash < 0) return url;
	return `${parsed.origin}${parsed.pathname.slice(0, slash + 1)}${variant}.jpg`;
}

// spotify rewrites a cover id to the smallest variant that still covers size,
// allowing the SLACK above. Namespaces it does not know are returned untouched:
// a wrong variant is a 404, which loses the art instead of just slowing it down.
function spotify(url: string, size: number): string {
	const parsed = new URL(url);
	const segments = parsed.pathname.split("/");
	const id = segments[segments.length - 1] ?? "";
	if (id.length !== 40) return url;
	const variants = SPOTIFY_VARIANTS[id.slice(0, 8)];
	if (!variants) return url;
	const pick =
		[...variants].reverse().find((entry) => entry.width >= size * SLACK) ??
		variants[0];
	segments[segments.length - 1] = id.slice(0, 12) + pick.code + id.slice(16);
	parsed.pathname = segments.join("/");
	return parsed.toString();
}

// cover rewrites a remote cover URL to the smallest variant the given display
// size needs: Spotify and Google art is resized by the Go /__img proxy, YouTube
// uses a smaller CDN variant directly. Unknown hosts are returned untouched, so
// this degrades to the original URL if anything is missing.
export function cover(
	url: string | undefined | null,
	size: number,
	android = isAndroid,
): string | undefined {
	if (!url) return url ?? undefined;
	const host = hostOf(url);
	if (!host) return url;
	if (YOUTUBE_HOSTS.has(host)) return youtube(url, size);
	if (!PROXY_HOSTS.has(host)) return url;
	const source = host === SPOTIFY_IMAGE ? spotify(url, size) : url;
	// The proxy answers through the asset handler, and the Android webview never
	// caches one of those: every launch would re-download and re-encode the whole
	// grid. The CDN already serves immutable art at the size picked above, and
	// the platform caches that.
	if (android) return source;
	return `/__img?w=${size}&u=${encodeURIComponent(source)}`;
}

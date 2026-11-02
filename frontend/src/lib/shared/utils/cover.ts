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

// cover rewrites a remote cover URL to the smallest variant the given display
// size needs: Spotify/Google art is resized by the Go /__img proxy, YouTube uses
// a smaller CDN variant directly. Unknown hosts are returned untouched, so this
// degrades to the original URL if anything is missing.
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
	// The proxy answers through the asset handler, and the Android webview never
	// caches one of those: every launch would re-download and re-encode the whole
	// grid. The CDN already serves immutable art, and the platform caches that.
	if (android) return url;
	return `/__img?w=${size}&u=${encodeURIComponent(url)}`;
}

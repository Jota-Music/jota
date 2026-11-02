import { expect, test } from "bun:test";
import { cover } from "@/lib/shared/utils/cover";

test("cover drops the signed query of a YouTube thumbnail", () => {
	expect(
		cover("https://i.ytimg.com/vi/abc/hq720.jpg?sqp=xyz&rs=AOn4", 384),
	).toBe("https://i.ytimg.com/vi/abc/hqdefault.jpg");
});

test("cover never asks for a YouTube variant that may not exist", () => {
	expect(cover("https://i.ytimg.com/vi/abc/mqdefault.jpg", 640)).toBe(
		"https://i.ytimg.com/vi/abc/hqdefault.jpg",
	);
	expect(cover("https://i.ytimg.com/vi/abc/mqdefault.jpg", 160)).toBe(
		"https://i.ytimg.com/vi/abc/mqdefault.jpg",
	);
});

test("cover keeps an unsigned YouTube thumbnail on the smaller variant", () => {
	expect(cover("https://i.ytimg.com/vi/abc/mqdefault.jpg", 80)).toBe(
		"https://i.ytimg.com/vi/abc/mqdefault.jpg",
	);
});

test("cover sends Spotify art through the image proxy", () => {
	expect(cover("https://i.scdn.co/image/abc", 80, false)).toBe(
		`/__img?w=80&u=${encodeURIComponent("https://i.scdn.co/image/abc")}`,
	);
});

// The proxy answers through the asset handler, and the Android webview never
// caches one of those: there the CDN is the cache.
test("cover leaves Spotify art to the CDN where the platform caches it", () => {
	expect(cover("https://i.scdn.co/image/abc", 80, true)).toBe(
		"https://i.scdn.co/image/abc",
	);
	expect(cover("https://mosaic.scdn.co/640/abc", 320, true)).toBe(
		"https://mosaic.scdn.co/640/abc",
	);
});

// The YouTube rewrite is a URL rewrite, not a proxy: it stays on every platform.
test("cover still asks YouTube for the smaller variant on Android", () => {
	expect(cover("https://i.ytimg.com/vi/abc/mqdefault.jpg", 160, true)).toBe(
		"https://i.ytimg.com/vi/abc/mqdefault.jpg",
	);
});

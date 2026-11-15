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

const ALBUM = "ab67616d0000b273e6926a5359f4eef19a3e97a3";
const ARTIST = "ab6761610000e5eb7e73f177ebe3f15a065fb939";

// Every id comes in 640, 300 and 64 flavours; a track row paints 80, so asking
// for the 640 one costs 77 KB and a 640px decode for nothing.
test("cover asks Spotify for the variant the size actually needs", () => {
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 80, true)).toBe(
		"https://i.scdn.co/image/ab67616d00001e02e6926a5359f4eef19a3e97a3",
	);
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 480, true)).toBe(
		`https://i.scdn.co/image/${ALBUM}`,
	);
	expect(cover(`https://i.scdn.co/image/${ARTIST}`, 128, true)).toBe(
		"https://i.scdn.co/image/ab6761610000f1787e73f177ebe3f15a065fb939",
	);
});

// A namespace whose variants were never measured stays as it is: a wrong
// variant answers 404, which loses the cover instead of slowing it down.
test("cover leaves an unknown Spotify namespace untouched", () => {
	const unknown =
		"https://i.scdn.co/image/deadbeef0000ffff0123456789abcdef01234567";
	expect(cover(unknown, 80, true)).toBe(unknown);
});

// 320 is the grid card and there is no 320 variant: taking the 300 one is 21 KB
// instead of 77 KB for art 6% narrower. 384 and 480 do need the 640.
test("cover takes the near variant for the grid, the big one for the player", () => {
	const near =
		"https://i.scdn.co/image/ab67616d00001e02e6926a5359f4eef19a3e97a3";
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 320, true)).toBe(near);
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 384, true)).toBe(
		`https://i.scdn.co/image/${ALBUM}`,
	);
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 480, true)).toBe(
		`https://i.scdn.co/image/${ALBUM}`,
	);
});

test("cover hands the smaller variant to the proxy on desktop", () => {
	expect(cover(`https://i.scdn.co/image/${ALBUM}`, 80, false)).toBe(
		`/__img?w=80&u=${encodeURIComponent("https://i.scdn.co/image/ab67616d00001e02e6926a5359f4eef19a3e97a3")}`,
	);
});

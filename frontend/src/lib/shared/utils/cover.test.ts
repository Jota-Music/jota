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
	expect(cover("https://i.scdn.co/image/abc", 80)).toBe(
		`/__img?w=80&u=${encodeURIComponent("https://i.scdn.co/image/abc")}`,
	);
});

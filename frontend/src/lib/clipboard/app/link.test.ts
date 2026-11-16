import { expect, test } from "bun:test";
import { clipboardLink } from "@/lib/clipboard/app/link";
import { roomLink } from "@/lib/sync/app/room-link";

const TRACK = "4cOdK2wGLETKBW3PvgPWqT";
const ALBUM = "4aawyAB9vmqN3uQ7FjRGTy";
const ARTIST = "4Z8W4fKeB5YxbusRsdQVPb";
const PLAYLIST = "37i9dQZF1DXcBWIGoYBM5M";
const VIDEO = "dQw4w9WgXcQ";
const LIST = "PL1234567890abcdef";

test("spotify links open their route", () => {
	expect(
		clipboardLink(`https://open.spotify.com/playlist/${PLAYLIST}`),
	).toEqual({
		kind: "playlist",
		text: `https://open.spotify.com/playlist/${PLAYLIST}`,
		to: `/playlist/${PLAYLIST}`,
	});
	expect(clipboardLink(`spotify:album:${ALBUM}`)).toMatchObject({
		kind: "album",
		to: `/album/${ALBUM}`,
	});
	expect(
		clipboardLink(`https://open.spotify.com/artist/${ARTIST}`),
	).toMatchObject({ kind: "artist", to: `/artist/${ARTIST}` });
});

test("a spotify track plays by id", () => {
	expect(clipboardLink(`spotify:track:${TRACK}`)).toMatchObject({
		kind: "song",
		to: TRACK,
	});
});

test("youtube links open their route or play the video", () => {
	expect(
		clipboardLink(`https://www.youtube.com/playlist?list=${LIST}`),
	).toEqual({
		kind: "playlist",
		text: `https://www.youtube.com/playlist?list=${LIST}`,
		to: `/playlist/youtube:${LIST}`,
	});
	expect(
		clipboardLink("https://www.youtube.com/channel/UC-lHJZR3Gqxm24_Vd_AJ5Yw"),
	).toMatchObject({
		kind: "channel",
		to: "/youtube/user/UC-lHJZR3Gqxm24_Vd_AJ5Yw",
	});
	expect(clipboardLink(`https://youtu.be/${VIDEO}`)).toMatchObject({
		kind: "song",
		to: `youtube:${VIDEO}`,
	});
});

test("a playlist parameter wins on the clipboard", () => {
	expect(
		clipboardLink(
			`https://www.youtube.com/watch?v=${VIDEO}&list=${LIST}&index=3`,
		),
	).toEqual({
		kind: "playlist",
		text: `https://www.youtube.com/watch?v=${VIDEO}&list=${LIST}&index=3`,
		to: `/playlist/youtube:${LIST}`,
	});
	expect(clipboardLink(`https://youtu.be/${VIDEO}?list=${LIST}`)).toEqual({
		kind: "playlist",
		text: `https://youtu.be/${VIDEO}?list=${LIST}`,
		to: `/playlist/youtube:${LIST}`,
	});
});

test("anything that is not a known link is ignored", () => {
	expect(clipboardLink("just some text")).toBeNull();
	expect(clipboardLink(`local:${TRACK}`)).toBeNull();
	expect(clipboardLink(`https://example.com/playlist/${PLAYLIST}`)).toBeNull();
	expect(clipboardLink("")).toBeNull();
});

test("bare ids and handles are search input, not clipboard links", () => {
	expect(clipboardLink(VIDEO)).toBeNull();
	expect(clipboardLink("aprendizaje")).toBeNull();
	expect(clipboardLink("@usuario")).toBeNull();
	expect(clipboardLink(`spotify:track:${TRACK}`)).not.toBeNull();
});

test("a room link carries its code, and the password when there is one", () => {
	expect(clipboardLink("room/consejo:1234")).toEqual({
		kind: "room",
		text: "room/consejo:1234",
		to: "/rooms",
		room: { code: "consejo", password: "1234" },
	});
	expect(clipboardLink("  room/consejo ")).toMatchObject({
		kind: "room",
		room: { code: "consejo", password: "" },
	});
	// A password is the tail, so a colon inside it is part of it.
	expect(clipboardLink("room/consejo:12:34")).toMatchObject({
		room: { code: "consejo", password: "12:34" },
	});
});

test("a room link with no code is not a room", () => {
	expect(clipboardLink("room/")).toBeNull();
	expect(clipboardLink("room")).toBeNull();
});

test("sharing a room writes what the paste detector reads back", () => {
	expect(roomLink("consejo", "1234")).toBe("room/consejo:1234");
	expect(roomLink("consejo", "")).toBe("room/consejo");
	expect(clipboardLink(roomLink("consejo", "1234"))).toMatchObject({
		kind: "room",
		room: { code: "consejo", password: "1234" },
	});
	expect(clipboardLink(roomLink("consejo", ""))).toMatchObject({
		room: { code: "consejo", password: "" },
	});
});

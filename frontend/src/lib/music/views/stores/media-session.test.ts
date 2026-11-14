import { expect, mock, test } from "bun:test";
import type { Song } from "@/lib/music/model";

const song: Song = {
	id: "4cOdK2wGLETKBW3PvgPWqT",
	url: "",
	name: "Rambo",
	duration: 200,
	share: { id: "4cOdK2wGLETKBW3PvgPWqT", url: "" },
	album: { id: "", title: "Album", url: "", covers: [] },
	artists: [],
};

test("pushes a seek at once instead of waiting out the throttle", async () => {
	const pushes: Array<{ position: number }> = [];
	mock.module("@bindings/app", () => ({
		UpdateDiscordPresence: async () => {},
		ClearDiscordPresence: async () => {},
	}));
	mock.module("@wailsio/runtime", () => ({ Events: { On: () => () => {} } }));
	const scope = globalThis as unknown as { window: unknown };
	const original = scope.window;
	scope.window = {
		wails: {
			mediaUpdate: (json: string) => pushes.push(JSON.parse(json)),
			mediaClear: () => {},
		},
	};

	try {
		const media = await import("@/lib/music/views/stores/media-session");
		const element = {
			currentTime: 0,
			duration: 200,
			playbackRate: 1,
		} as HTMLAudioElement;

		media.update(song, true);
		media.position(element);
		const settled = pushes.length;

		element.currentTime = 120;
		media.position(element);

		expect(pushes).toHaveLength(settled + 1);
		expect(pushes.at(-1)?.position).toBe(120);

		// The system moves before the element lands, and a drift correction
		// small enough to be invisible leaves it alone.
		media.seek(150);
		expect(pushes).toHaveLength(settled + 2);
		expect(pushes.at(-1)?.position).toBe(150);

		media.seek(150.2);
		expect(pushes).toHaveLength(settled + 2);
	} finally {
		scope.window = original;
	}
});

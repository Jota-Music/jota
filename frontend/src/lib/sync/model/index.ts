import type { ControlAction, Song } from "@/lib/music/model";

export type Seat = "off" | "host" | "guest";

// What identifies a track in the shared queue, plus the room cosmetics. Sent by
// the member that announces the next track.
export interface Track {
	songId: string;
	song?: Song;
	youtubeId?: string;
	index: number;
	shuffle?: boolean;
	repeat?: "off" | "all" | "one";
	color?: string | null;
	binary?: string | null;
}

// A track plus its playback. `at` is stamped by the relay, never by the member,
// so every clock comparison uses the relay clock.
export interface Playback extends Track {
	playing: boolean;
	positionMs: number;
	at?: number;
}

export type ClientMessage =
	| { t: "join"; at: number }
	| { t: "ping"; id: number; at: number }
	| ({ t: "control" } & ControlAction)
	| { t: "queue"; data: string }
	| ({ t: "state" } & Playback)
	| ({ t: "sync" } & Playback)
	| ({ t: "prepare"; gen: string; epoch: string } & Track)
	| { t: "ready"; gen: string; ok: boolean }
	| { t: "snapshot"; to: string; state: Playback };

export type ServerMessage =
	| { t: "role"; role: "host" | "guest" }
	| { t: "members"; count: number; epoch: string }
	| { t: "pong"; id: number; at: number; echo: number }
	| { t: "play"; gen: string; epoch: string; at: number; positionMs: number }
	| ({ t: "sync" } & Playback)
	| {
			t: "snapshot";
			// Relay-clock stamps of the join exchange (t1 receive, t2 send).
			echo: number;
			at: number;
			queue?: Song[];
			// The host's live playback, or the relay's cached state on fallback.
			state?: Playback;
	  }
	| { t: "join"; at: number; t1: number; from: string }
	| { t: "queue"; data: string }
	| ({ t: "control" } & ControlAction)
	| ({ t: "prepare"; gen: string; epoch: string } & Track)
	| { t: "error"; reason: string };

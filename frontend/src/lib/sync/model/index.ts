import type { Song } from "@/lib/music/model";

export type ControlAction =
	| { action: "toggle" }
	| { action: "seek"; positionMs: number }
	| { action: "next" }
	| { action: "prev" }
	| { action: "shuffle" }
	| { action: "repeat" }
	| { action: "play"; index: number }
	| { action: "enqueue"; song: Song }
	| { action: "playSelection"; songId: string; songs: Song[] }
	| { action: "remove"; index: number }
	| { action: "move"; from: number; to: number }
	| { action: "moveAfter"; index: number };

export type PeerMessage =
	| {
			t: "state";
			at: number;
			playing: boolean;
			positionMs: number;
			songId: string;
			youtubeId?: string;
			index: number;
			shuffle?: boolean;
			repeat?: "off" | "all" | "one";
			color?: string | null;
			binary?: string | null;
	  }
	| { t: "queue"; data: string }
	| {
			t: "heartbeat";
			at: number;
			playing: boolean;
			positionMs: number;
			songId: string;
			color?: string | null;
			binary?: string | null;
	  }
	| { t: "ping"; id: number; at: number }
	| { t: "pong"; id: number; at: number; echo: number }
	| { t: "members"; count: number }
	| { t: "role"; role: "host" | "guest" }
	| { t: "prepare"; songId: string; youtubeId?: string }
	| { t: "ready"; songId: string }
	| ({ t: "control" } & ControlAction)
	| { t: "error"; reason: string };

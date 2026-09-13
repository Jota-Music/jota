export type Role = "off" | "host" | "guest";

export type PeerMessage =
	| {
			t: "state";
			at: number;
			playing: boolean;
			positionMs: number;
			songId: string;
			youtubeId?: string;
			index: number;
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
	| { t: "error"; reason: string };

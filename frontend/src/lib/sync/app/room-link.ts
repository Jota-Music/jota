export type RoomLink = {
	code: string;
	password: string;
};

const PREFIX = "room/";

export function roomLink(code: string, password: string): string {
	const name = code.trim();
	const secret = password.trim();
	return secret ? `${PREFIX}${name}:${secret}` : `${PREFIX}${name}`;
}

// A room travels as `room/<code>[:<password>]`, so a copied line is enough to
// hear it. The code is one segment; the password is whatever follows, colons
// included, because it is the tail and nothing else is parsed after it.
export function parseRoomLink(text: string): RoomLink | null {
	const trimmed = text.trim();
	if (!trimmed.startsWith(PREFIX)) return null;
	const [code, ...rest] = trimmed.slice(PREFIX.length).split(":");
	const name = code?.trim();
	if (!name) return null;
	return { code: name, password: rest.join(":").trim() };
}

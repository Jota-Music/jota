import { t } from "@/lib/shared/i18n";

// Backend (Go/relay) errors cross the wire as plain English messages. Mapping
// them to keys keeps the UI translatable without changing the Go contracts.
const BACKEND: Record<string, string> = {
	"In-app browser login is not available on this platform":
		"errors.backend.loginUnavailable",
	"login window closed": "errors.backend.loginWindowClosed",
	"login timed out": "errors.backend.loginTimedOut",
	"empty cookies": "errors.backend.emptyCookies",
	"cookies missing SAPISID": "errors.backend.missingSapisid",
	"the browser's cookies could not be decrypted: unlock the keyring and try again":
		"errors.backend.keyringLocked",
	"the YouTube session cannot open private resources: import a cookies.txt export":
		"errors.backend.sessionPartial",
	"update: install already in progress": "errors.backend.installInProgress",
	"update: automatic install is not supported on this channel":
		"errors.backend.installUnsupported",
	"host left": "errors.relay.hostLeft",
	"room is full": "errors.relay.roomFull",
	"room already has a host": "errors.relay.roomHasHost",
	"invalid room password": "errors.relay.badPassword",
	"relay unreachable": "errors.relay.unreachable",
	"every recovery attempt failed": "errors.recoveryFailed",
	"the browser needs a click to allow playback": "errors.playbackBlocked",
	"consensus timeout": "errors.consensusTimeout",
	"the backup file was written by another version":
		"settings.autosave.errors.version",
	"the backup file cannot be read": "settings.autosave.errors.unreadable",
	"cipher: message authentication failed": "auth.youtube.badPass",
};

export function translateError(error: unknown): string {
	const message = error instanceof Error ? error.message : String(error);
	// A binding rejection is an Error whose String() carries the class name
	// ("RuntimeError: room is full"), so the map is consulted without it too:
	// the name is an implementation detail the user should never read.
	const bare = message.replace(/^\w*(Error|Exception): /, "");
	return translated(bare) ?? bare;
}

// A Go error reaches the page wrapped in whichever walk carried it ("Binding
// call failed: <method>: <reason>"), so every suffix is tried and the inner
// reason is the part worth translating.
function translated(message: string): string | undefined {
	const parts = message.split(": ");
	for (let i = 0; i < parts.length; i++) {
		const key = BACKEND[parts.slice(i).join(": ")];
		if (key) return t(key);
	}
	return undefined;
}

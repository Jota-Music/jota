import { t } from ".";

// Backend (Go/relay) errors cross the wire as plain English messages. Mapping
// them to keys keeps the UI translatable without changing the Go contracts.
const BACKEND: Record<string, string> = {
	"In-app browser login is not available on this platform":
		"errors.backend.loginUnavailable",
	"login window closed": "errors.backend.loginWindowClosed",
	"login timed out": "errors.backend.loginTimedOut",
	"empty cookies": "errors.backend.emptyCookies",
	"cookies missing SAPISID": "errors.backend.missingSapisid",
	"update: install already in progress": "errors.backend.installInProgress",
	"update: automatic install is not supported on this channel":
		"errors.backend.installUnsupported",
	"host left": "errors.relay.hostLeft",
	"room is full": "errors.relay.roomFull",
	"room already has a host": "errors.relay.roomHasHost",
	"invalid room password": "errors.relay.badPassword",
	"every recovery attempt failed": "errors.recoveryFailed",
	"the browser needs a click to allow playback": "errors.playbackBlocked",
	"consensus timeout": "errors.consensusTimeout",
};

export function translateError(error: unknown): string {
	const message = error instanceof Error ? error.message : String(error);
	const key = BACKEND[message];
	return key ? t(key) : message;
}

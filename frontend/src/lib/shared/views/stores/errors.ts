import { LogError } from "@bindings/app";
import { signal } from "@preact/signals";
import { translateError } from "@/lib/shared/i18n/errors";

type RoomError = {
	id: string;
	message: string;
	tone: "error" | "info";
	leaving?: boolean;
};

export const roomErrors = signal<RoomError[]>([]);

// An error is read, not noticed: it stays twice as long as a confirmation.
const LIFETIME = { error: 8000, info: 4000 } as const;
// Must match the notif-out keyframe: the card is gone from the store once it has
// finished leaving, not while it is still visible.
const EXIT = 160;

const timers = new Map<string, ReturnType<typeof setTimeout>>();

function cancel(id: string): void {
	const timer = timers.get(id);
	if (timer === undefined) return;
	clearTimeout(timer);
	timers.delete(id);
}

function detail(error: unknown): string {
	if (typeof DOMException !== "undefined" && error instanceof DOMException) {
		return `${error.name}: ${error.message}`;
	}
	if (error instanceof Error) {
		// WebKit's stack is a list of frames with no message on it, so the message
		// has to be written next to it or the log says only where it failed.
		return `${error.name}: ${error.message}\n${error.stack ?? ""}`;
	}
	if (typeof error === "string") return error;
	try {
		return JSON.stringify(error);
	} catch {
		return String(error);
	}
}

function line(context: string | undefined, text: string): string {
	return context ? `${context}: ${text}` : text;
}

// Log only. Use for failures the app already recovers from (skip/retry/expired
// URL), so the log keeps the full detail without spamming the UI.
export function logError(error: unknown, context?: string): void {
	try {
		void LogError(line(context, detail(error))).catch(() => {});
	} catch {
		// No backend (e.g. the frontend served outside Wails).
	}
}

function push(entry: RoomError): void {
	roomErrors.value = [...roomErrors.value, entry];

	// Keep at most three visible notifications; fade the rest out.
	if (roomErrors.value.length > 3) {
		removeError(roomErrors.value[0].id);
	}

	timers.set(
		entry.id,
		setTimeout(() => {
			removeError(entry.id);
		}, LIFETIME[entry.tone]),
	);
}

// Log and surface in the error bar.
export function addError(error: unknown, context?: string): void {
	logError(error, context);

	push({
		id: `${Date.now()}-${Math.random()}`,
		message: line(context, translateError(error)),
		tone: "error",
	});
}

// Surface a confirmation in the same bar, without logging.
export function notify(message: string): void {
	push({ id: `${Date.now()}-${Math.random()}`, message, tone: "info" });
}

export function removeError(id: string): void {
	const target = roomErrors.value.find((e) => e.id === id);
	if (!target || target.leaving) return;
	cancel(id);
	roomErrors.value = roomErrors.value.map((e) =>
		e.id === id ? { ...e, leaving: true } : e,
	);
	setTimeout(() => {
		roomErrors.value = roomErrors.value.filter((e) => e.id !== id);
	}, EXIT);
}

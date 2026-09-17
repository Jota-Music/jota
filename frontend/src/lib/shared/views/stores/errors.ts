import { LogError } from "@bindings/app";
import { signal } from "@preact/signals";
import { translateError } from "@/lib/shared/i18n/errors";

type RoomError = {
	id: string;
	message: string;
	timestamp: number;
};

export const roomErrors = signal<RoomError[]>([]);

function detail(error: unknown): string {
	if (typeof DOMException !== "undefined" && error instanceof DOMException) {
		return `${error.name}: ${error.message}`;
	}
	if (error instanceof Error) return error.stack ?? error.message;
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

// Log and surface in the error bar.
export function addError(error: unknown, context?: string): void {
	logError(error, context);

	const entry: RoomError = {
		id: `${Date.now()}-${Math.random()}`,
		message: line(context, translateError(error)),
		timestamp: Date.now(),
	};

	roomErrors.value = [...roomErrors.value, entry];

	// Auto-remove after 5 seconds
	setTimeout(() => {
		removeError(entry.id);
	}, 5000);
}

export function removeError(id: string): void {
	roomErrors.value = roomErrors.value.filter((e) => e.id !== id);
}

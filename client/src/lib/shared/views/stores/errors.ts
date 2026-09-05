import { signal } from "@preact/signals";

export type RoomError = {
	id: string;
	message: string;
	timestamp: number;
};

export const roomErrors = signal<RoomError[]>([]);

export function addError(message: string): void {
	const error: RoomError = {
		id: `${Date.now()}-${Math.random()}`,
		message,
		timestamp: Date.now(),
	};

	roomErrors.value = [...roomErrors.value, error];

	// Auto-remove after 5 seconds
	setTimeout(() => {
		removeError(error.id);
	}, 5000);
}

export function removeError(id: string): void {
	roomErrors.value = roomErrors.value.filter((e) => e.id !== id);
}

export function clearErrors(): void {
	roomErrors.value = [];
}

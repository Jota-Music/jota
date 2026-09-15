import { signal } from "@preact/signals";

const KEY = "welcome:dismissed";

export const welcomeOpen = signal(false);

export function welcomeDismissed(): boolean {
	return localStorage.getItem(KEY) === "1";
}

export function dismissWelcome(forever: boolean): void {
	welcomeOpen.value = false;
	if (forever) localStorage.setItem(KEY, "1");
}

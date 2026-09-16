import { signal } from "@preact/signals";

const KEY = "welcome:dismissed";

let dismissed = false;

export const welcomeOpen = signal(false);

export function welcomeDismissed(): boolean {
	return dismissed || localStorage.getItem(KEY) === "1";
}

export function dismissWelcome(forever: boolean): void {
	dismissed = true;
	welcomeOpen.value = false;
	if (forever) localStorage.setItem(KEY, "1");
}

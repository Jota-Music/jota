import { effect, signal } from "@preact/signals";

const STORAGE_KEY = "followed_users";

function loadFollowedUsers(): string[] {
	if (typeof window === "undefined") return [];
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		return Array.isArray(parsed)
			? parsed.filter((u): u is string => typeof u === "string")
			: [];
	} catch {
		return [];
	}
}

export const followedUsers = signal<string[]>(loadFollowedUsers());

effect(() => {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(followedUsers.value));
});

export function toggleFollow(user: string): void {
	if (followedUsers.value.includes(user)) {
		followedUsers.value = followedUsers.value.filter((u) => u !== user);
	} else {
		followedUsers.value = [...followedUsers.value, user];
	}
}

export function isFollowing(user: string): boolean {
	return followedUsers.value.includes(user);
}

import { FollowUser, GetFollowedUsers, UnfollowUser } from "@bindings/app";

// A follow is named the way every id is: the platform and kind, then the id the
// source answers. Older stores and documents wrote `artist:<id>`,
// `youtube:<channel>` and a user by the bare handle, so both are read.
export function artistFollow(id: string): string {
	return `spotify:artist:${id.trim()}`;
}

export function channelFollow(id: string): string {
	return `youtube:channel:${id.trim().replace(/^youtube:(channel:)?/, "")}`;
}

export function userFollow(handle: string): string {
	return `spotify:user:${handle
		.trim()
		.replace(/^spotify:user:/, "")
		.replace(/^@/, "")}`;
}

export type FollowTarget = {
	// key is the id the following shelf and its order use.
	key: string;
	artist?: string;
	channel?: string;
	user?: string;
};

export function parseFollow(id: string): FollowTarget {
	const value = id.trim();
	if (value.startsWith("youtube:")) {
		const channel = value.replace(/^youtube:(channel:)?/, "");
		return { key: channelFollow(channel), channel };
	}
	if (value.startsWith("artist:") || value.startsWith("spotify:artist:")) {
		const artist = value.slice(value.lastIndexOf(":") + 1);
		return { key: artistFollow(artist), artist };
	}
	if (value.startsWith("user:") || value.startsWith("spotify:user:")) {
		const user = value.replace(/^(spotify:)?user:/, "");
		return { key: userFollow(user), user };
	}
	return { key: userFollow(value), user: value };
}

export async function getFollowedUsers(): Promise<string[]> {
	return (await GetFollowedUsers()) ?? [];
}

export async function followUser(user: string): Promise<void> {
	await FollowUser(user);
}

export async function unfollowUser(user: string): Promise<void> {
	await UnfollowUser(user);
}

import { FollowUser, GetFollowedUsers, UnfollowUser } from "@bindings/app";

export async function getFollowedUsers(): Promise<string[]> {
	return (await GetFollowedUsers()) ?? [];
}

export async function followUser(user: string): Promise<void> {
	await FollowUser(user);
}

export async function unfollowUser(user: string): Promise<void> {
	await UnfollowUser(user);
}

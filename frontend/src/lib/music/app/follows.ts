import { FollowUser, GetFollowedUsers, UnfollowUser } from "@bindings/app";

export async function getFollowedUsers(account: string): Promise<string[]> {
	const users = (await GetFollowedUsers(account)) as unknown as string[] | null;
	return users ?? [];
}

export async function followUser(account: string, user: string): Promise<void> {
	await FollowUser(account, user);
}

export async function unfollowUser(
	account: string,
	user: string,
): Promise<void> {
	await UnfollowUser(account, user);
}

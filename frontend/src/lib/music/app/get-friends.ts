import { GetFriends } from "@bindings/app";

export default async function getFriends(): Promise<string[]> {
	const users = (await GetFriends()) as unknown as string[] | null;
	return users ?? [];
}

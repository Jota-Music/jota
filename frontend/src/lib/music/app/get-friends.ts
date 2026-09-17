import { GetFriends } from "@bindings/app";

export default async function getFriends(): Promise<string[]> {
	return (await GetFriends()) ?? [];
}

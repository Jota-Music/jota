import { GetPlaylistOrder, SavePlaylistOrder } from "@bindings/app";

export async function getOrder(account: string): Promise<string[]> {
	return (await GetPlaylistOrder(account)) ?? [];
}

export async function saveOrder(account: string, ids: string[]): Promise<void> {
	await SavePlaylistOrder(account, ids);
}

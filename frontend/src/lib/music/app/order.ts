import { GetPlaylistOrder, SavePlaylistOrder } from "@bindings/app";

export async function getOrder(account: string): Promise<string[]> {
	const ids = (await GetPlaylistOrder(account)) as unknown as string[] | null;
	return ids ?? [];
}

export async function saveOrder(account: string, ids: string[]): Promise<void> {
	await SavePlaylistOrder(account, ids);
}

import { GetOrder, SaveOrder } from "@bindings/app";

export async function getOrder(
	scope: string,
	account: string,
): Promise<string[]> {
	return (await GetOrder(scope, account)) ?? [];
}

export async function saveOrder(
	scope: string,
	account: string,
	ids: string[],
): Promise<void> {
	await SaveOrder(scope, account, ids);
}

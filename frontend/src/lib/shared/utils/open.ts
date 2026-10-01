import { LogError, OpenURL } from "@bindings/app";

async function attempt(url: string): Promise<void> {
	try {
		await OpenURL(url);
	} catch (error) {
		await LogError(`open ${url}: ${error}`);
		throw error;
	}
}

// Rethrows so a caller waiting on a login can bail out instead of sitting in a
// pending state until the backend timeout: without a browser the redirect never
// happens and the wait looks like a hang.
export function open(url: string): Promise<void> {
	return attempt(url);
}

// For links where there is nothing to wait for and nothing to retry, so a
// missing browser opener must not surface as an unhandled rejection.
export function browse(url: string): void {
	void attempt(url).catch(() => {});
}

import { LogError, OpenURL } from "@bindings/app";

export function open(url: string): Promise<void> {
	return OpenURL(url).catch((error) => LogError(`open ${url}: ${error}`));
}

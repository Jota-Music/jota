import { OpenURL } from "@bindings/app";
import { Browser } from "@wailsio/runtime";

export function open(url: string): Promise<void> {
	if (navigator.userAgent.toLowerCase().includes("android")) {
		return OpenURL(url);
	}
	return Browser.OpenURL(url);
}

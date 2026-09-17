import { effect, signal } from "@preact/signals";
import {
	type Catalog,
	format,
	lookup,
	type Params,
} from "@/lib/shared/i18n/translate";

const BASE = "en";
const STORAGE_KEY = "locale";

const modules = import.meta.glob("./locales/*.json", {
	eager: true,
	import: "default",
}) as Record<string, Catalog>;

const messages: Record<string, Catalog> = {};
for (const [path, data] of Object.entries(modules)) {
	messages[path.slice(path.lastIndexOf("/") + 1, path.lastIndexOf("."))] = data;
}

export const available = Object.keys(messages).sort();

function detect(): string {
	if (typeof window === "undefined") return BASE;
	const saved = localStorage.getItem(STORAGE_KEY);
	if (saved && messages[saved]) return saved;
	const nav = (navigator.language || BASE).slice(0, 2).toLowerCase();
	return messages[nav] ? nav : BASE;
}

export const locale = signal(detect());

export function setLocale(code: string): void {
	if (messages[code]) locale.value = code;
}

export function t(key: string, params: Params = {}): string {
	const entry =
		lookup(messages[locale.value] ?? {}, key) ??
		lookup(messages[BASE] ?? {}, key);
	return entry === undefined ? key : format(entry, locale.value, params);
}

effect(() => {
	const code = locale.value;
	if (typeof window === "undefined") return;
	localStorage.setItem(STORAGE_KEY, code);
	document.documentElement.lang = code;
});

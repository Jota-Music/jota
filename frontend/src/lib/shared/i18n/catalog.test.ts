import { expect, test } from "bun:test";
import { Glob } from "bun";
import en from "@/lib/shared/i18n/locales/en.json";
import es from "@/lib/shared/i18n/locales/es.json";
import { lookup } from "@/lib/shared/i18n/translate";

// Every `t("...")` in the app is a literal key, so a key a view asks for and a
// catalog does not hold renders the key itself ("sync.rooms.connect" on a
// button). Keys built from a template are not checked here: they are only known
// at run time.
const KEY = /\bt\(\s*"([a-z0-9_.]+)"/g;
const SOURCES = new Glob("src/**/*.{ts,tsx}");

test("every key the app asks for is in both catalogs", async () => {
	const missing: string[] = [];
	for await (const file of SOURCES.scan(".")) {
		if (file.includes("/locales/") || file.endsWith(".test.ts")) continue;
		const text = await Bun.file(file).text();
		for (const [, key] of text.matchAll(KEY)) {
			const gone = [es, en].some(
				(catalog) => lookup(catalog, key) === undefined,
			);
			if (gone) missing.push(`${key} (${file})`);
		}
	}
	expect(missing).toEqual([]);
});

import { expect, test } from "bun:test";
import { format, interpolate, lookup } from "@/lib/shared/i18n/translate";

test("lookup resolves nested keys and misses undefined", () => {
	const catalog = { a: { b: { c: "value" } } };
	expect(lookup(catalog, "a.b.c")).toBe("value");
	expect(lookup(catalog, "a.b.z")).toBeUndefined();
	expect(lookup(catalog, "a.z.c")).toBeUndefined();
});

test("interpolate fills placeholders and keeps unknown ones", () => {
	expect(interpolate("{count} tracks", { count: 3 })).toBe("3 tracks");
	expect(interpolate("{a} {b}", { a: "x" })).toBe("x {b}");
});

test("format picks the plural category per locale", () => {
	const en = { one: "{count} track", other: "{count} tracks" };
	const es = { one: "{count} canción", other: "{count} canciones" };
	expect(format(en, "en", { count: 1 })).toBe("1 track");
	expect(format(en, "en", { count: 2 })).toBe("2 tracks");
	expect(format(es, "es", { count: 1 })).toBe("1 canción");
	expect(format(es, "es", { count: 5 })).toBe("5 canciones");
});

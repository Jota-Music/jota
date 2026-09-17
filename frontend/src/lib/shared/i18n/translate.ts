export type Catalog = Record<string, unknown>;
export type Entry = string | Record<string, string>;
export type Params = Record<string, string | number>;

export function lookup(catalog: Catalog, key: string): Entry | undefined {
	const value = key
		.split(".")
		.reduce<unknown>(
			(acc, part) =>
				acc && typeof acc === "object"
					? (acc as Record<string, unknown>)[part]
					: undefined,
			catalog,
		);
	return typeof value === "string" || (value && typeof value === "object")
		? (value as Entry)
		: undefined;
}

export function interpolate(template: string, params: Params): string {
	return template.replace(/\{(\w+)\}/g, (_, name: string) =>
		params[name] === undefined ? `{${name}}` : String(params[name]),
	);
}

export function format(entry: Entry, locale: string, params: Params): string {
	if (typeof entry === "string") return interpolate(entry, params);
	const count = Number(params.count ?? 0);
	const category = new Intl.PluralRules(locale).select(count);
	const template = entry[category] ?? entry.other ?? entry.one;
	return template ? interpolate(template, params) : "";
}

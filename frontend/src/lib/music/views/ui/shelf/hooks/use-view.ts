import { useState } from "preact/hooks";
import type { SourceFilter, Variant } from "../types";

export const defaultViewKey = "cover_grid_view";
const filterKey = "playlist_source_filter";

function loadVariant(key: string): Variant {
	return localStorage.getItem(key) === "compact" ? "compact" : "grid";
}

function loadFilter(): SourceFilter {
	const saved = localStorage.getItem(filterKey);
	return saved === "spotify" || saved === "youtube" || saved === "local"
		? saved
		: "all";
}

export function useView(viewKey: string) {
	const [variant, setVariantState] = useState<Variant>(() =>
		loadVariant(viewKey),
	);
	const [filter, setFilterState] = useState<SourceFilter>(loadFilter);

	const setVariant = (next: Variant) => {
		setVariantState(next);
		localStorage.setItem(viewKey, next);
	};

	const setFilter = (next: SourceFilter) => {
		setFilterState(next);
		localStorage.setItem(filterKey, next);
	};

	return { variant, filter, setVariant, setFilter };
}

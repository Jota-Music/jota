import { useState } from "preact/hooks";
import type { SourceFilter, Variant } from "../types";

export const defaultViewKey = "cover_grid_view";
const filterKey = (viewKey: string) => `${viewKey}_source_filter`;

function loadVariant(key: string): Variant {
	return localStorage.getItem(key) === "compact" ? "compact" : "grid";
}

function loadFilter(viewKey: string): SourceFilter {
	const saved = localStorage.getItem(filterKey(viewKey));
	return saved === "spotify" || saved === "youtube" || saved === "local"
		? saved
		: "all";
}

export function useView(viewKey: string) {
	const [variant, setVariantState] = useState<Variant>(() =>
		loadVariant(viewKey),
	);
	const [filter, setFilterState] = useState<SourceFilter>(() =>
		loadFilter(viewKey),
	);

	const setVariant = (next: Variant) => {
		setVariantState(next);
		localStorage.setItem(viewKey, next);
	};

	const setFilter = (next: SourceFilter) => {
		setFilterState(next);
		localStorage.setItem(filterKey(viewKey), next);
	};

	return { variant, filter, setVariant, setFilter };
}

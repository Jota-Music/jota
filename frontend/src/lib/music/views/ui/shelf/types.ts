import type { ComponentChildren, ComponentType } from "preact";
import type { Action } from "@/lib/shared/views/ui/components/context-menu";

export type Variant = "grid" | "compact";
export type SourceFilter = "all" | "spotify" | "youtube" | "local";
export type Source = "spotify" | "youtube" | "local";

export type IconType = ComponentType<{ size?: number | string }>;

export interface Item {
	id: string;
	name: string;
	cover?: string;
	covers?: string[];
	subtitle?: ComponentChildren;
	source?: Source;
	removable?: boolean;
	icon?: IconType;
	// placeholder fills the cover slot when there is no cover, taking over from
	// `icon`. It receives the size the slot expects.
	placeholder?: (size: number) => ComponentChildren;
	// overlay renders on top of the cover slot, cover or not. It receives the
	// size the slot expects.
	overlay?: (size: number) => ComponentChildren;
	// menu adds item-specific actions to the context menu, before Remove.
	menu?: Action[];
}

export interface Props {
	items: Item[];
	to: (id: string) => string;
	// viewKey scopes the grid/compact choice to one shelf, so each can keep its
	// own layout.
	viewKey?: string;
	onSelect?: (id: string) => void;
	// onPlay adds a "Play" action to each item's context menu when set.
	onPlay?: (id: string) => void;
	// actions renders hover buttons for an item, next to the remove button.
	actions?: (id: string) => ComponentChildren;
	isLoading?: boolean;
	emptyMessage?: ComponentChildren;
	onRemove?: (id: string) => void;
	onReorder?: (fromId: string, toId: string) => void;
	// onCreate renders a "new playlist" button next to the source filters.
	onCreate?: () => void;
	// showLocal hides the "Local" source filter when the shelf has no local items.
	showLocal?: boolean;
}

import { signal } from "@preact/signals";
import { EllipsisVertical, type LucideIcon } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useLayoutEffect, useRef } from "preact/hooks";
import { bottomBarHeight } from "@/lib/shared/utils/layout";
import { cn } from "@/lib/shared/utils/tw";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";

export type Action = {
	icon: LucideIcon;
	label: string;
	danger?: boolean;
	// on is a toggle whose state is already set, so its icon reads as pressed:
	// the same red, filled mark the heart shows outside the menu.
	on?: boolean;
	run: () => void;
};

type Request = {
	actions: Action[];
	x: number;
	y: number;
	header?: string;
	// Clearance below the viewport bottom for a fixed element anchored there
	// (the compact player bar): the y clamp keeps the menu above it instead of
	// covering its controls.
	bottom: number;
	anchor: Element | null;
};

const GAP = 6;

// Without a mouse there is nothing to point at, so touch gets the same sheet the
// modals use instead of a floating menu.
const COARSE = window.matchMedia("(pointer: coarse)").matches;

const request = signal<Request | null>(null);

export function openContextMenu(
	actions: Action[],
	e: MouseEvent,
	header?: string,
	bottom = 0,
) {
	if (actions.length === 0) return;
	e.preventDefault();
	const anchor = (e.currentTarget as Element | null) ?? null;
	if (e.type === "click" && anchor && request.value?.anchor === anchor) {
		request.value = null;
		return;
	}
	request.value = {
		actions: [...actions].sort(
			(a, b) => Number(!!a.danger) - Number(!!b.danger),
		),
		x: e.clientX,
		y: e.clientY,
		header,
		bottom,
		anchor,
	};
}

export function closeContextMenu() {
	request.value = null;
}

type MenuButtonProps = {
	actions: Action[];
	// label names the button for a screen reader, since the trigger is an icon.
	label: string;
	// header labels the menu itself when it is about a selection rather than
	// about one item.
	header?: string;
	// reserve keeps the button in place while the actions are still loading, so
	// the header it sits in does not shift when they land.
	reserve?: boolean;
	// bottom is the clearance below the viewport bottom for a fixed element
	// anchored there (the compact player bar).
	bottom?: number;
};

// MenuButton is the ⋯ trigger for a menu built elsewhere: a page header whose
// actions are about the page, or a row that reuses the menu its list builds.
export function MenuButton({
	actions,
	label,
	header,
	reserve = false,
	bottom = 0,
}: MenuButtonProps) {
	const empty = actions.length === 0;
	if (empty && !reserve) return null;
	return (
		<button
			type="button"
			title={label}
			aria-label={label}
			aria-haspopup="menu"
			disabled={empty}
			onPointerDown={(event) => {
				event.preventDefault();
				event.stopPropagation();
			}}
			onClick={(event) => {
				event.stopPropagation();
				openContextMenu(actions, event, header, bottom);
			}}
			class="shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:text-white disabled:cursor-default disabled:opacity-40"
		>
			<EllipsisVertical size={18} />
		</button>
	);
}

const item = (action: Action, index: number) => (
	<button
		// The label is not unique (the same action can appear twice in a menu),
		// and a duplicate key makes Preact reuse the wrong node.
		key={`${index}-${action.label}`}
		type="button"
		role={COARSE ? undefined : "menuitem"}
		onClick={() => {
			closeContextMenu();
			action.run();
		}}
		class={cn(
			"flex w-full cursor-pointer items-center gap-3 text-left text-balance transition-colors",
			COARSE ? "px-4 py-3.5 text-base" : "px-4 py-2.5 text-sm",
			action.danger
				? "text-red-400 hover:bg-red-950/40"
				: "text-zinc-300 hover:bg-zinc-800/80 hover:text-white",
		)}
	>
		<action.icon
			size={16}
			class={cn(
				action.danger || action.on ? "text-red-400" : "text-zinc-500",
				action.on && "fill-current",
			)}
		/>
		{action.label}
	</button>
);

const header = (menu: Request) =>
	menu.header && (
		<>
			<div class="px-4 pt-2 pb-1 text-left text-balance text-xs font-medium text-zinc-500">
				{menu.header}
			</div>
			<div class="mx-2 mb-1 border-t border-zinc-800" />
		</>
	);

function FloatingMenu() {
	const menu = request.value;
	const ref = useRef<HTMLDivElement>(null);

	useLayoutEffect(() => {
		if (!menu) return;
		const el = ref.current;
		if (!el) return;

		// ponytail: measure the fixed bottom bar (h-14 + safe-area inset) so
		// the menu never sits behind it on phones with a gesture bar.
		const bar = bottomBarHeight();
		const x = Math.max(
			GAP,
			Math.min(menu.x, window.innerWidth - el.offsetWidth - GAP),
		);
		const y = Math.max(
			GAP,
			Math.min(
				menu.y,
				window.innerHeight - bar - menu.bottom - el.offsetHeight - GAP,
			),
		);
		el.style.left = `${x}px`;
		el.style.top = `${y}px`;
		// The menu takes the focus, so closing it has to give it back: a keyboard
		// user who dismissed it with Escape was left with nothing focused.
		const restore = document.activeElement as HTMLElement | null;
		el.querySelector("button")?.focus();

		const onDown = (e: PointerEvent) => {
			if (el.contains(e.target as Node)) return;
			// The trigger toggles the menu on click: closing here would
			// reopen it a moment later.
			if (menu.anchor?.contains(e.target as Node)) return;
			closeContextMenu();
		};
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape") closeContextMenu();
		};
		const onScroll = () => closeContextMenu();
		const onResize = () => closeContextMenu();
		document.addEventListener("pointerdown", onDown, true);
		document.addEventListener("keydown", onKey);
		window.addEventListener("scroll", onScroll, true);
		window.addEventListener("resize", onResize);
		return () => {
			document.removeEventListener("pointerdown", onDown, true);
			document.removeEventListener("keydown", onKey);
			window.removeEventListener("scroll", onScroll, true);
			window.removeEventListener("resize", onResize);
			// Only when the menu is what had the focus: a pointer that opened it
			// elsewhere keeps the focus where the click landed.
			if (el.contains(document.activeElement)) {
				const back = (menu.anchor ?? restore) as HTMLElement | null;
				back?.focus();
			}
		};
	}, [menu]);

	if (!menu) return null;

	return createPortal(
		<div
			ref={ref}
			role="menu"
			class="fixed z-300 w-56 max-h-[calc(100dvh-0.75rem)] overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950 py-1 shadow-xl"
			style={{ top: 0, left: 0 }}
		>
			{header(menu)}
			{menu.actions.map((action, index) => item(action, index))}
		</div>,
		document.body,
	);
}

function ActionSheet() {
	const menu = request.value;
	return (
		<Sheet open={!!menu} close={closeContextMenu} centered={false}>
			{menu && header(menu)}
			{/* A phone holds fewer actions than a playlist menu has: the list
			    scrolls inside the sheet instead of running under the player bar. */}
			<div class="min-h-0 overflow-y-auto">{menu?.actions.map(item)}</div>
		</Sheet>
	);
}

export function ContextMenu() {
	const menu = request.value;
	return menu && COARSE ? <ActionSheet /> : <FloatingMenu />;
}

export default ContextMenu;

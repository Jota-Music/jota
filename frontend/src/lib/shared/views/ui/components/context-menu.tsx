import { signal } from "@preact/signals";
import type { LucideIcon } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useLayoutEffect, useRef } from "preact/hooks";
import { bottomBarHeight } from "@/lib/shared/utils/layout";
import { cn } from "@/lib/shared/utils/tw";
import { Sheet } from "@/lib/shared/views/ui/components/sheet";

export type Action = {
	icon: LucideIcon;
	label: string;
	danger?: boolean;
	run: () => void;
};

type Request = {
	actions: Action[];
	x: number;
	y: number;
	header?: string;
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
) {
	if (actions.length === 0) return;
	e.preventDefault();
	request.value = { actions, x: e.clientX, y: e.clientY, header };
}

export function closeContextMenu() {
	request.value = null;
}

const item = (action: Action) => (
	<button
		key={action.label}
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
			class={action.danger ? "text-red-400" : "text-zinc-500"}
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
			Math.min(menu.y, window.innerHeight - bar - el.offsetHeight - GAP),
		);
		el.style.left = `${x}px`;
		el.style.top = `${y}px`;
		el.querySelector("button")?.focus();

		const onDown = (e: PointerEvent) => {
			if (!el.contains(e.target as Node)) closeContextMenu();
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
		};
	}, [menu]);

	if (!menu) return null;

	return createPortal(
		<div
			ref={ref}
			role="menu"
			class="fixed z-50 w-56 rounded-lg border border-zinc-800 bg-zinc-950 py-1 shadow-xl"
			style={{ top: 0, left: 0 }}
		>
			{header(menu)}
			{menu.actions.map(item)}
		</div>,
		document.body,
	);
}

function ActionSheet() {
	const menu = request.value;
	return (
		<Sheet open={!!menu} close={closeContextMenu} centered={false}>
			{menu && header(menu)}
			{menu?.actions.map(item)}
		</Sheet>
	);
}

export function ContextMenu() {
	return COARSE ? <ActionSheet /> : <FloatingMenu />;
}

export default ContextMenu;

import { signal } from "@preact/signals";
import type { LucideIcon } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useLayoutEffect, useRef } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

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
};

const GAP = 6;

const request = signal<Request | null>(null);

export function openContextMenu(actions: Action[], e: MouseEvent) {
	if (actions.length === 0) return;
	e.preventDefault();
	request.value = { actions, x: e.clientX, y: e.clientY };
}

export function closeContextMenu() {
	request.value = null;
}

export function ContextMenu() {
	const menu = request.value;
	const ref = useRef<HTMLDivElement>(null);

	useLayoutEffect(() => {
		if (!menu) return;
		const el = ref.current;
		if (!el) return;

		// ponytail: mobile bottom bar is fixed h-14, so the menu never fits
		// below it; 56px constant instead of measuring the bar.
		const bar = window.matchMedia("(min-width: 768px)").matches ? 0 : 56;
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
			{menu.actions.map((action) => (
				<button
					key={action.label}
					type="button"
					role="menuitem"
					onClick={() => {
						closeContextMenu();
						action.run();
					}}
					class={cn(
						"flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-left text-balance text-sm transition-colors",
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
			))}
		</div>,
		document.body,
	);
}

export default ContextMenu;

import { type ComponentChildren, createContext, type RefObject } from "preact";
import { createPortal } from "preact/compat";
import { useContext, useEffect, useRef, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

export const OverlayHost = createContext<RefObject<HTMLDivElement> | null>(
	null,
);

interface SheetProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	mobileOnly?: boolean;
	children: ComponentChildren;
}

const EXIT_MS = 300;
const CLOSE_PX = 120;
const FLING_PX_PER_MS = 0.5;

export function Sheet({
	open,
	close,
	labelledBy,
	closeLabel = "Close",
	mobileOnly = false,
	children,
}: SheetProps) {
	const [mounted, setMounted] = useState(open);
	const [shown, setShown] = useState(false);
	const panelRef = useRef<HTMLDivElement>(null);
	const drag = useRef({ active: false, startY: 0, startTime: 0, offset: 0 });
	const host = useContext(OverlayHost)?.current ?? null;

	const clearInline = () => {
		const el = panelRef.current;
		if (!el) return;
		el.style.transition = "";
		el.style.translate = "";
	};

	useEffect(() => {
		if (open) {
			setMounted(true);
			clearInline();
			const id = requestAnimationFrame(() => setShown(true));
			return () => cancelAnimationFrame(id);
		}
		setShown(false);
		const id = setTimeout(() => setMounted(false), EXIT_MS);
		return () => clearTimeout(id);
	}, [open]);

	if (!open && !mounted) return null;

	const onDragStart = (e: PointerEvent) => {
		if (!window.matchMedia("(max-width: 639px)").matches) return;
		drag.current = {
			active: true,
			startY: e.clientY,
			startTime: performance.now(),
			offset: 0,
		};
		const el = panelRef.current;
		if (el) el.style.transition = "none";
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	};

	const onDragMove = (e: PointerEvent) => {
		if (!drag.current.active) return;
		const delta = e.clientY - drag.current.startY;
		drag.current.offset = delta > 0 ? delta : 0;
		const el = panelRef.current;
		if (el) el.style.translate = `0 ${drag.current.offset}px`;
	};

	const onDragEnd = () => {
		if (!drag.current.active) return;
		drag.current.active = false;
		const elapsed = performance.now() - drag.current.startTime;
		const velocity = drag.current.offset / Math.max(elapsed, 1);
		const el = panelRef.current;
		if (drag.current.offset > CLOSE_PX || velocity > FLING_PX_PER_MS) {
			if (el) {
				el.style.transition = "translate 0.25s ease-out";
				el.style.translate = "0 100%";
			}
			close();
		} else if (el) {
			el.style.transition = "translate 0.3s cubic-bezier(0.32, 0.72, 0, 1)";
			el.style.translate = "0 0";
		}
		drag.current.offset = 0;
	};

	return createPortal(
		<div
			class={cn(
				"z-50 flex items-end justify-center p-0 sm:items-center sm:p-4",
				host ? "absolute inset-0" : "fixed inset-0",
				mobileOnly && "md:hidden",
			)}
			role="presentation"
		>
			<button
				type="button"
				class={cn(
					"absolute inset-0 bg-black/70 transition-opacity duration-300",
					shown ? "opacity-100" : "opacity-0",
				)}
				aria-label={closeLabel}
				onClick={close}
			/>

			<div
				ref={panelRef}
				onTransitionEnd={(e) => {
					if (e.propertyName === "translate") clearInline();
				}}
				class={cn(
					"relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-zinc-800 bg-zinc-950 pb-[env(safe-area-inset-bottom)] shadow-2xl transition-[translate,scale,opacity] duration-300 ease-out sm:rounded-2xl",
					host ? "max-h-full" : "max-h-[85dvh]",
					shown
						? "translate-y-0 opacity-100 sm:scale-100"
						: "translate-y-full opacity-0 sm:translate-y-0 sm:scale-95",
				)}
				role="dialog"
				aria-modal="true"
				aria-labelledby={labelledBy}
			>
				<div
					class="flex shrink-0 cursor-grab touch-none justify-center pt-2 pb-1 active:cursor-grabbing sm:hidden"
					onPointerDown={onDragStart}
					onPointerMove={onDragMove}
					onPointerUp={onDragEnd}
					onPointerCancel={onDragEnd}
					aria-hidden
				>
					<div class="h-1.5 w-10 rounded-full bg-zinc-700" />
				</div>
				{children}
			</div>
		</div>,
		host ?? document.body,
	);
}

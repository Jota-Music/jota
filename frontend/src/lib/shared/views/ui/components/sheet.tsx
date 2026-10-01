import { useSignal } from "@preact/signals";
import { X } from "lucide-preact";
import { type ComponentChildren, createContext, type RefObject } from "preact";
import { createPortal } from "preact/compat";
import { useContext, useEffect, useRef } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { useSheetDrag } from "@/lib/shared/views/hooks/use-sheet-drag";
import { hold } from "@/lib/shared/views/stores/overlays";

export const OverlayHost = createContext<RefObject<HTMLDivElement> | null>(
	null,
);

interface SheetProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	// abovePlayer anchors the sheet above the collapsed player bar so the bar
	// stays visible; false lets the sheet cover it (full player).
	abovePlayer?: boolean;
	mobileOnly?: boolean;
	// centered turns the sheet into a centered dialog from sm up. A touch
	// action sheet stays anchored to the bottom at every width, since a coarse
	// pointer can be a tablet.
	centered?: boolean;
	children: ComponentChildren;
}

export function CloseButton({
	close,
	label = t("common.close"),
	class: className,
}: {
	close: () => void;
	label?: string;
	class?: string;
}) {
	return (
		<button
			type="button"
			class={cn(
				"rounded-lg p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer",
				className,
			)}
			aria-label={label}
			onClick={close}
		>
			<X size={20} />
		</button>
	);
}

const EXIT_MS = 300;

export function Sheet({
	open,
	close,
	labelledBy,
	closeLabel = t("common.close"),
	abovePlayer = true,
	mobileOnly = false,
	centered = true,
	children,
}: SheetProps) {
	const mounted = useSignal(open);
	const shown = useSignal(false);
	const panelRef = useRef<HTMLDivElement>(null);
	const backdropRef = useRef<HTMLButtonElement>(null);
	const host = useContext(OverlayHost)?.current ?? null;
	const dragging = useSheetDrag(panelRef, backdropRef, close, mounted.value);

	// Most call sites pass an inline arrow, so `close` changes identity on
	// every render; the overlay stack needs the live one, not the first.
	const closeRef = useRef(close);
	closeRef.current = close;

	useEffect(() => {
		if (!open) return;
		return hold(() => closeRef.current());
	}, [open]);

	const clearInline = () => {
		const el = panelRef.current;
		if (!el) return;
		el.style.transition = "";
		el.style.translate = "";
	};

	useEffect(() => {
		if (open) {
			mounted.value = true;
			clearInline();
			const id = requestAnimationFrame(() => {
				shown.value = true;
			});
			return () => cancelAnimationFrame(id);
		}
		shown.value = false;
		const id = setTimeout(() => {
			mounted.value = false;
		}, EXIT_MS);
		return () => clearTimeout(id);
	}, [open]);

	if (!open && !mounted.value) return null;

	return createPortal(
		<div
			class={cn(
				"z-50 md:z-120 flex items-end justify-center p-0",
				centered && "sm:items-center sm:p-4",
				host ? "absolute inset-0" : "fixed inset-0",
				mobileOnly && "md:hidden",
			)}
			role="presentation"
		>
			<button
				ref={backdropRef}
				type="button"
				class={cn(
					"absolute inset-x-0 top-0 bg-black/70 transition-opacity duration-300 sm:inset-0",
					abovePlayer
						? "bottom-[calc(3.5rem+var(--player-compact)+env(safe-area-inset-bottom))]"
						: "bottom-0",
					shown.value ? "opacity-100" : "opacity-0",
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
					"relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-b-0 border-zinc-800 bg-zinc-950 pb-3 shadow-2xl transition-[translate,scale,opacity] duration-300 ease-out",
					centered && "sm:pb-0 sm:rounded-2xl sm:border-b sm:mb-0",
					abovePlayer
						? "mb-[calc(3.5rem+var(--player-compact)+env(safe-area-inset-bottom))]"
						: "mb-[calc(3.5rem+env(safe-area-inset-bottom))]",
					"max-h-[70dvh]",
					host ? "sm:max-h-full" : "sm:max-h-[85dvh]",
					shown.value
						? centered
							? "translate-y-0 opacity-100 sm:scale-100"
							: "translate-y-0 opacity-100"
						: centered
							? "translate-y-[calc(100%+3.5rem)] opacity-100 sm:translate-y-0 sm:opacity-0 sm:scale-95"
							: "translate-y-[calc(100%+3.5rem)] opacity-100",
				)}
				role="dialog"
				aria-modal="true"
				aria-labelledby={labelledBy}
			>
				{/* A sheet is mostly buttons, so the handle strip is its only grab
				    zone and gets room to be caught. A centered dialog keeps the
				    tight strip, or the full player opens with a gaping top. */}
				<div
					class={cn(
						"flex shrink-0 touch-none justify-center",
						centered ? "pb-2 pt-3 sm:hidden" : "py-4",
					)}
					aria-hidden
				>
					<div
						class={cn(
							"h-1.5 w-12 rounded-full transition-[scale,background-color] duration-150",
							dragging ? "scale-x-125 bg-zinc-400" : "bg-zinc-600",
						)}
					/>
				</div>

				<CloseButton
					close={close}
					label={closeLabel}
					class="absolute right-3 top-3 z-20"
				/>

				{children}
			</div>
		</div>,
		host ?? document.body,
	);
}

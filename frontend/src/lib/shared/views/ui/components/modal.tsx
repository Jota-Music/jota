import { X } from "lucide-preact";
import { type ComponentChildren, createContext, type RefObject } from "preact";
import { createPortal } from "preact/compat";
import { useContext, useEffect, useRef, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";
import { useSheetDrag } from "@/lib/shared/views/ui/hooks/use-sheet-drag";

export const OverlayHost = createContext<RefObject<HTMLDivElement> | null>(
	null,
);

interface ModalProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	mobileOnly?: boolean;
	children: ComponentChildren;
}

const EXIT_MS = 300;

export function Modal({
	open,
	close,
	labelledBy,
	closeLabel = "Close",
	mobileOnly = false,
	children,
}: ModalProps) {
	const [mounted, setMounted] = useState(open);
	const [shown, setShown] = useState(false);
	const panelRef = useRef<HTMLDivElement>(null);
	const backdropRef = useRef<HTMLButtonElement>(null);
	const host = useContext(OverlayHost)?.current ?? null;
	const dragging = useSheetDrag(panelRef, backdropRef, close, mounted);

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
				ref={backdropRef}
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
					"relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-b-0 border-zinc-800 bg-zinc-950 pb-[env(safe-area-inset-bottom)] shadow-2xl transition-[translate,scale,opacity] duration-300 ease-out sm:rounded-2xl sm:border-b",
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
					class="flex shrink-0 touch-none justify-center pb-2 pt-3 sm:hidden"
					aria-hidden
				>
					<div
						class={cn(
							"h-1.5 w-12 rounded-full transition-[scale,background-color] duration-150",
							dragging ? "scale-x-125 bg-zinc-400" : "bg-zinc-600",
						)}
					/>
				</div>

				<button
					type="button"
					class="absolute right-2 top-2 z-20 rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer"
					aria-label={closeLabel}
					onClick={close}
				>
					<X size={20} />
				</button>

				{children}
			</div>
		</div>,
		host ?? document.body,
	);
}

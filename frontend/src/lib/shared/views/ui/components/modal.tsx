import { useSignal } from "@preact/signals";
import { X } from "lucide-preact";
import { type ComponentChildren, createContext, type RefObject } from "preact";
import { createPortal } from "preact/compat";
import { useContext, useEffect, useRef } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { useSheetDrag } from "@/lib/shared/views/hooks/use-sheet-drag";

export const OverlayHost = createContext<RefObject<HTMLDivElement> | null>(
	null,
);

interface ModalProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	mobileOnly?: boolean;
	hideClose?: boolean;
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

export function ModalHeader({
	children,
	close,
	closeLabel,
	bordered = true,
}: {
	children?: ComponentChildren;
	close: () => void;
	closeLabel?: string;
	bordered?: boolean;
}) {
	return (
		<header
			class={cn(
				"flex shrink-0 items-center justify-between gap-3 px-4 py-3",
				bordered && "border-b border-zinc-800",
			)}
		>
			<div class="flex min-w-0 items-center gap-2 pl-1.5">{children}</div>
			<CloseButton close={close} label={closeLabel} />
		</header>
	);
}

const EXIT_MS = 300;

export function Modal({
	open,
	close,
	labelledBy,
	closeLabel = t("common.close"),
	mobileOnly = false,
	hideClose = false,
	children,
}: ModalProps) {
	const mounted = useSignal(open);
	const shown = useSignal(false);
	const panelRef = useRef<HTMLDivElement>(null);
	const backdropRef = useRef<HTMLButtonElement>(null);
	const host = useContext(OverlayHost)?.current ?? null;
	const dragging = useSheetDrag(panelRef, backdropRef, close, mounted.value);

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
					"relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-b-0 border-zinc-800 bg-zinc-950 pb-[env(safe-area-inset-bottom)] shadow-2xl transition-[translate,scale,opacity] duration-300 ease-out sm:rounded-2xl sm:border-b",
					"max-h-[70dvh]",
					host ? "sm:max-h-full" : "sm:max-h-[85dvh]",
					shown.value
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

				{!hideClose && (
					<CloseButton
						close={close}
						label={closeLabel}
						class="absolute right-2 top-1.5 z-20"
					/>
				)}

				{children}
			</div>
		</div>,
		host ?? document.body,
	);
}

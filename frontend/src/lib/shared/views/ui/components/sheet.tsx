import type { ComponentChildren } from "preact";
import { createPortal } from "preact/compat";
import { useEffect, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

interface SheetProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	children: ComponentChildren;
}

const EXIT_MS = 300;

export function Sheet({
	open,
	close,
	labelledBy,
	closeLabel = "Cerrar",
	children,
}: SheetProps) {
	const [mounted, setMounted] = useState(open);
	const [shown, setShown] = useState(false);

	useEffect(() => {
		if (open) {
			setMounted(true);
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
			class="fixed inset-0 z-50 flex items-end justify-center p-0 sm:items-center sm:p-4"
			role="presentation"
		>
			<button
				type="button"
				class={cn(
					"absolute inset-0 bg-black/70 backdrop-blur-[2px] transition-opacity duration-300",
					shown ? "opacity-100" : "opacity-0",
				)}
				aria-label={closeLabel}
				onClick={close}
			/>

			<div
				class={cn(
					"relative z-10 flex h-[80dvh] w-full max-w-2xl flex-col overflow-hidden rounded-t-2xl border border-zinc-800 bg-zinc-950 shadow-2xl transition-[translate,scale,opacity] duration-300 ease-out sm:rounded-2xl",
					shown
						? "translate-y-0 opacity-100 sm:scale-100"
						: "translate-y-full opacity-0 sm:translate-y-0 sm:scale-95",
				)}
				role="dialog"
				aria-modal="true"
				aria-labelledby={labelledBy}
			>
				{children}
			</div>
		</div>,
		document.body,
	);
}

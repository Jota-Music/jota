import { X } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect, useRef, useState } from "preact/hooks";
import * as store from "@/lib/p2p/views/stores";
import { QRCode } from "@/lib/p2p/views/ui/qrcode";
import { cn } from "@/lib/shared/utils/tw";

const EXIT_MS = 200;

export function QrModal() {
	const value = store.qrValue.value;
	const [mounted, setMounted] = useState(!!value);
	const [shown, setShown] = useState(false);
	const last = useRef(value);
	if (value) last.current = value;

	useEffect(() => {
		if (value) {
			setMounted(true);
			const id = requestAnimationFrame(() => setShown(true));
			return () => cancelAnimationFrame(id);
		}
		setShown(false);
		const id = setTimeout(() => setMounted(false), EXIT_MS);
		return () => clearTimeout(id);
	}, [value]);

	if (!mounted) return null;
	const qr = value ?? last.current ?? "";

	return createPortal(
		<div
			class="fixed inset-0 z-[60] flex items-center justify-center p-4"
			role="presentation"
		>
			<button
				type="button"
				class={cn(
					"absolute inset-0 bg-black/80 backdrop-blur-sm transition-opacity duration-200",
					shown ? "opacity-100" : "opacity-0",
				)}
				aria-label="Cerrar QR"
				onClick={() => (store.qrValue.value = null)}
			/>
			<div
				class={cn(
					"relative z-10 w-full max-w-lg rounded-2xl border border-zinc-800 bg-zinc-950 p-4 shadow-2xl transition-[translate,scale,opacity] duration-200 ease-out",
					shown
						? "translate-y-0 scale-100 opacity-100"
						: "translate-y-4 scale-95 opacity-0",
				)}
				role="dialog"
				aria-modal="true"
				aria-label="QR code"
			>
				<div class="mb-3 flex items-center justify-between">
					<span class="text-sm text-zinc-300">Escanea este código</span>
					<button
						type="button"
						class="rounded-lg p-1.5 text-zinc-400 transition hover:bg-zinc-800 hover:text-white cursor-pointer"
						aria-label="Cerrar"
						onClick={() => (store.qrValue.value = null)}
					>
						<X size={20} />
					</button>
				</div>
				<QRCode
					value={qr}
					class="aspect-square w-full rounded-lg bg-white p-3"
				/>
			</div>
		</div>,
		document.body,
	);
}

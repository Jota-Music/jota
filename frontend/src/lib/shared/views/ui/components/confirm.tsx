import { Check, X } from "lucide-preact";
import { useRef } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { Modal } from "@/lib/shared/views/ui/components/modal";

export function ConfirmModal({
	open,
	danger = false,
	pending = false,
	close,
	onConfirm,
}: {
	open: boolean;
	danger?: boolean;
	pending?: boolean;
	close: () => void;
	onConfirm: () => void;
}) {
	// Keep the look frozen while the modal animates out: the parent clears the
	// target on close, which would otherwise flip the button colour mid-exit.
	const look = useRef({ danger, pending });
	if (open) look.current = { danger, pending };

	return (
		<Modal open={open} close={close} labelledBy="confirm-title">
			<div class="flex flex-col items-center gap-5 p-6 pt-10 sm:pt-6">
				<p id="confirm-title" class="text-center text-sm text-zinc-200">
					{t("common.confirm")}
				</p>
				<div class="flex items-center gap-3">
					<button
						type="button"
						title={t("common.cancel")}
						aria-label={t("common.cancel")}
						onClick={close}
						class="flex size-11 cursor-pointer items-center justify-center rounded-full border border-zinc-800 text-zinc-300 transition-colors hover:bg-zinc-800 hover:text-white"
					>
						<X size={20} />
					</button>
					<button
						type="button"
						title={t("common.confirmAction")}
						aria-label={t("common.confirmAction")}
						disabled={look.current.pending}
						onClick={onConfirm}
						class={cn(
							"flex size-11 cursor-pointer items-center justify-center rounded-full transition-colors disabled:opacity-40",
							look.current.danger
								? "bg-red-900/50 text-red-200 hover:bg-red-900/70"
								: "bg-(--dominant-color) text-(--binary-color) hover:opacity-80",
						)}
					>
						<Check size={20} />
					</button>
				</div>
			</div>
		</Modal>
	);
}

export default ConfirmModal;

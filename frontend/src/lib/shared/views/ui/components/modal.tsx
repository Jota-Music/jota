import type { ComponentChildren } from "preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import {
	CloseButton,
	OverlayHost,
	Sheet,
} from "@/lib/shared/views/ui/components/sheet";

export { CloseButton, OverlayHost };

// The close button belongs to the sheet, not to the header: it always sits in
// the same top-right corner. A header is only the row of content under it, so it
// reserves that corner instead of running under it.
export function ModalHeader({
	children,
	bordered = true,
}: {
	children?: ComponentChildren;
	bordered?: boolean;
}) {
	return (
		<header
			class={cn(
				"flex shrink-0 items-center gap-2 py-3 pl-5.5 pr-12",
				bordered && "border-b border-zinc-800",
			)}
		>
			{children}
		</header>
	);
}

interface ModalProps {
	open: boolean;
	close: () => void;
	labelledBy?: string;
	closeLabel?: string;
	mobileOnly?: boolean;
	children: ComponentChildren;
}

export function Modal({
	open,
	close,
	labelledBy,
	closeLabel = t("common.close"),
	mobileOnly = false,
	children,
}: ModalProps) {
	return (
		<Sheet
			open={open}
			close={close}
			labelledBy={labelledBy}
			closeLabel={closeLabel}
			mobileOnly={mobileOnly}
		>
			{children}
		</Sheet>
	);
}

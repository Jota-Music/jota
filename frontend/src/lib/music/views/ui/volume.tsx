// VolumeControl.tsx

import { useSignal } from "@preact/signals";
import { Volume, Volume1, Volume2, VolumeX } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";
import {
	muted,
	setVolume,
	toggleMute,
	volume,
} from "@/lib/music/views/stores/audio";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import Progress from "@/lib/shared/views/ui/components/progress";

const VOLUME_STATUS = {
	MUTED: VolumeX,
	LOW: Volume,
	MEDIUM: Volume1,
	HIGH: Volume2,
};

const getVolumeStatus = (value: number, muted: boolean) => {
	if (muted) return VOLUME_STATUS.MUTED;
	if (value === 0) return VOLUME_STATUS.MUTED;
	if (value < 1 / 3) return VOLUME_STATUS.LOW;
	if (value < 2 / 3) return VOLUME_STATUS.MEDIUM;
	return VOLUME_STATUS.HIGH;
};

// Space between the icon and its panel, and the grace a pointer gets to cross
// it: enough to travel, short enough that the panel does not linger. The open
// delay keeps a pointer on its way to the play button from flashing it.
const TIP_GAP = 8;
const TIP_LEASE = 200;
const TIP_OPEN = 120;

function VolumeControl({
	class: className,
	compact = false,
	tip = false,
}: {
	class?: string;
	compact?: boolean;
	// No room for a bar per control (the compact player): the slider hangs above
	// the icon on hover instead. A finger has no hover, so touch keeps the icon
	// alone and its click still mutes.
	tip?: boolean;
}) {
	const Icon = getVolumeStatus(volume.value, muted.value);
	const inline = !tip;

	const open = useSignal(false);
	const anchor = useRef<HTMLButtonElement>(null);
	const panel = useRef<HTMLDivElement>(null);
	const lease = useRef<number | null>(null);

	const cancelClose = () => {
		if (lease.current === null) return;
		clearTimeout(lease.current);
		lease.current = null;
	};

	const closeSoon = () => {
		cancelClose();
		lease.current = window.setTimeout(() => {
			lease.current = null;
			open.value = false;
		}, TIP_LEASE);
	};

	const openSoon = () => {
		cancelClose();
		lease.current = window.setTimeout(() => {
			lease.current = null;
			open.value = true;
		}, TIP_OPEN);
	};

	useEffect(() => cancelClose, []);

	useLayoutEffect(() => {
		if (!open.value) return;
		const a = anchor.current;
		const p = panel.current;
		if (!a || !p) return;
		// Laid out straight on the element: a signal here would paint the panel at
		// the corner for a frame before the measurement landed.
		const rect = a.getBoundingClientRect();
		const left = Math.max(
			TIP_GAP,
			Math.min(
				rect.left + rect.width / 2 - p.offsetWidth / 2,
				window.innerWidth - p.offsetWidth - TIP_GAP,
			),
		);
		p.style.left = `${left}px`;
		p.style.top = `${rect.top - p.offsetHeight - TIP_GAP}px`;
	}, [open.value]);

	useEffect(() => {
		if (!open.value) return;
		const key = (e: KeyboardEvent) => {
			if (e.key === "Escape") open.value = false;
		};
		document.addEventListener("keydown", key);
		return () => document.removeEventListener("keydown", key);
	}, [open.value]);

	return (
		<div
			class={cn(
				"flex items-center gap-2 text-(--dominant-color)",
				inline && (compact ? "w-24" : "w-26 md:w-36"),
				className,
			)}
		>
			<button
				ref={anchor}
				type="button"
				onClick={toggleMute}
				onPointerEnter={(e) => {
					if (!tip || e.pointerType !== "mouse") return;
					openSoon();
				}}
				onPointerLeave={(e) => {
					if (!tip || e.pointerType !== "mouse") return;
					closeSoon();
				}}
				// A pointer is not the only way in: the keyboard opens the same panel,
				// and the slider inside it is reachable from there.
				onFocus={() => {
					if (tip) openSoon();
				}}
				onBlur={() => {
					if (tip) closeSoon();
				}}
				aria-label={
					muted.value ? t("music.volume.unmute") : t("music.volume.mute")
				}
				class={
					inline
						? // display:contents keeps the icon in the row, so the focus mark
							// has to land on the icon itself
							"contents focus-visible:[&>svg]:rounded focus-visible:[&>svg]:ring-2 focus-visible:[&>svg]:ring-current"
						: "rounded-md p-1.5 cursor-pointer"
				}
			>
				<Icon class="size-6 cursor-pointer" />
			</button>

			{inline && (
				<Progress
					class="min-w-0 flex-1 text-current"
					label={t("music.volume.label")}
					value={volume.value}
					onChange={setVolume}
				/>
			)}

			{tip &&
				open.value &&
				createPortal(
					<div
						ref={panel}
						onPointerEnter={cancelClose}
						onPointerLeave={closeSoon}
						class="fixed z-300 rounded-lg border border-zinc-800 bg-zinc-950 px-2.5 py-3 text-(--dominant-color) shadow-xl"
						style={{ left: 0, top: 0 }}
					>
						<Progress
							vertical
							class="text-current"
							label={t("music.volume.label")}
							value={volume.value}
							onChange={setVolume}
						/>
					</div>,
					document.body,
				)}
		</div>
	);
}

export default VolumeControl;

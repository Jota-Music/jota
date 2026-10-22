// VolumeControl.tsx

import { Volume, Volume1, Volume2, VolumeX } from "lucide-preact";
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

function VolumeControl({
	class: className,
	compact = false,
}: {
	class?: string;
	compact?: boolean;
}) {
	const Icon = getVolumeStatus(volume.value, muted.value);
	return (
		<div
			class={cn(
				"flex items-center gap-2 text-(--dominant-color)",
				compact ? "w-24" : "w-26 md:w-36",
				className,
			)}
		>
			<button
				type="button"
				onClick={toggleMute}
				aria-label={
					muted.value ? t("music.volume.unmute") : t("music.volume.mute")
				}
				class="contents"
			>
				<Icon
					class={compact ? "size-5 cursor-pointer" : "size-9 cursor-pointer"}
				/>
			</button>

			<Progress
				class={compact ? "w-16 text-current" : "w-36 text-current"}
				value={volume.value}
				onChange={setVolume}
			/>
		</div>
	);
}

export default VolumeControl;

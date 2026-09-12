// VolumeControl.tsx

import { Volume, Volume1, Volume2, VolumeX } from "lucide-preact";
import {
	muted,
	setVolume,
	toggleMute,
	volume,
} from "@/lib/music/views/stores/audio";
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
	if (value < 0.5) return VOLUME_STATUS.LOW;
	if (value < 0.75) return VOLUME_STATUS.MEDIUM;
	return VOLUME_STATUS.HIGH;
};

function VolumeControl({
	class: _class,
	className,
}: {
	class?: string;
	className?: string;
}) {
	const Icon = getVolumeStatus(volume.value, muted.value);
	return (
		<div class={cn("w-36 flex items-center gap-4", _class, className)}>
			<Icon onClick={toggleMute} size={35} class="cursor-pointer" />

			<Progress
				class="w-36 text-current"
				value={volume.value}
				onChange={setVolume}
			/>
		</div>
	);
}

export default VolumeControl;

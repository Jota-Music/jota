import {
	moveAfterCurrent,
	moveQueue,
	playAt,
	unqueue,
} from "@/lib/music/views/stores/player";
import { currentIndex, queue, showQueue } from "@/lib/music/views/stores/queue";

export function useQueuePanel() {
	return {
		open: showQueue.value,
		songs: queue.value,
		currentIndex: currentIndex.value,
		removeFromQueueAt: unqueue,
		moveQueueItem: moveQueue,
		moveJustBelowCurrentPlaying: moveAfterCurrent,
		playSongAtQueueIndex: playAt,
	};
}

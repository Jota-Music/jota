import { Check, ListMusic } from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { queuePulse, showQueue } from "@/lib/music/views/stores/queue";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

const FEEDBACK_MS = 1000;

export default function QueueButton({ class: className }: { class?: string }) {
	const [added, setAdded] = useState(false);
	const initial = useRef(true);

	useEffect(() => {
		if (initial.current) {
			initial.current = false;
			return;
		}
		setAdded(true);
		const timer = setTimeout(() => setAdded(false), FEEDBACK_MS);
		return () => clearTimeout(timer);
	}, [queuePulse.value]);

	return (
		<button
			type="button"
			title={t("music.player.queue")}
			aria-expanded={showQueue.value}
			onClick={() => {
				showQueue.value = !showQueue.value;
			}}
			class={cn(
				"rounded-md p-1.5 transition text-white/60 hover:text-white cursor-pointer",
				className,
			)}
		>
			<span class="grid place-items-center">
				<ListMusic
					class={`col-start-1 row-start-1 size-5 fill-current transition-all duration-300 ease-out ${added ? "scale-50 opacity-0" : "scale-100 opacity-100"}`}
				/>
				<Check
					class={`col-start-1 row-start-1 size-5 stroke-current text-(--dominant-color) transition-all duration-300 ease-out ${added ? "scale-100 opacity-100" : "scale-50 opacity-0"}`}
				/>
			</span>
		</button>
	);
}

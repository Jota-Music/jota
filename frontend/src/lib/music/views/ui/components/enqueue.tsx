import { Check, ListPlus } from "lucide-preact";
import { useRef, useState } from "preact/hooks";
import type { Song } from "@/lib/music/model";
import { enqueue } from "@/lib/music/views/stores/player";
import { cn } from "@/lib/shared/utils/tw";

const FEEDBACK_MS = 1000;

export default function Enqueue({
	song,
	title,
	class: className,
}: {
	song: Song;
	title: string;
	class?: string;
}) {
	const [added, setAdded] = useState(false);
	const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

	return (
		<button
			type="button"
			title={title}
			onClick={(e) => {
				e.stopPropagation();
				enqueue(song);
				setAdded(true);
				if (timer.current) clearTimeout(timer.current);
				timer.current = setTimeout(() => setAdded(false), FEEDBACK_MS);
			}}
			class={cn(
				"shrink-0 cursor-pointer rounded-md p-1.5 text-zinc-400 transition hover:bg-zinc-800/80 hover:text-amber-300",
				className,
			)}
		>
			<span class="grid place-items-center">
				<ListPlus
					size={18}
					strokeWidth={2}
					class={`col-start-1 row-start-1 transition-all duration-300 ease-out ${added ? "scale-50 opacity-0" : "scale-100 opacity-100"}`}
				/>
				<Check
					size={18}
					strokeWidth={2}
					class={`col-start-1 row-start-1 text-amber-300 transition-all duration-300 ease-out ${added ? "scale-100 opacity-100" : "scale-50 opacity-0"}`}
				/>
			</span>
		</button>
	);
}

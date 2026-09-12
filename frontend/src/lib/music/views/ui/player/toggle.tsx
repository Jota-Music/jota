import { Loader, Pause, Play } from "lucide-preact";
import { cn } from "@/lib/shared/utils/tw";

export function Toggle(props: {
	loading: boolean;
	playing: boolean;
	onClick: () => void;
	size: number;
	class?: string;
}) {
	const { loading, playing, onClick, size, class: className } = props;
	return (
		<button
			type="button"
			onClick={onClick}
			class={cn(
				"p-3 rounded-full transition-all drop-shadow-lg drop-shadow-black cursor-pointer",
				className,
			)}
		>
			<div class="flex items-center justify-center">
				{loading ? (
					<Loader size={size} class="animate-spin text-(--binary-color)" />
				) : playing ? (
					<Pause size={size} class="fill-current text-(--binary-color)" />
				) : (
					<Play size={size} class="fill-current text-(--binary-color)" />
				)}
			</div>
		</button>
	);
}

export default Toggle;

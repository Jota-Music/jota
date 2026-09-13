import { Loader, Pause, Play } from "lucide-preact";
import { cn } from "@/lib/shared/utils/tw";

export function Toggle(props: {
	loading: boolean;
	playing: boolean;
	onClick: () => void;
	iconClass?: string;
	class?: string;
}) {
	const { loading, playing, onClick, iconClass, class: className } = props;
	const icon = cn("fill-current text-(--binary-color)", iconClass);
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
					<Loader class={cn(icon, "animate-spin")} />
				) : playing ? (
					<Pause class={icon} />
				) : (
					<Play class={icon} />
				)}
			</div>
		</button>
	);
}

export default Toggle;

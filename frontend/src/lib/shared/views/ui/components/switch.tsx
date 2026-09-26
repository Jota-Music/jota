import { cn } from "@/lib/shared/utils/tw";

type Props = {
	checked: boolean;
	label: string;
	disabled?: boolean;
	onToggle: () => void;
};

export function Switch({ checked, label, disabled, onToggle }: Props) {
	return (
		<button
			type="button"
			role="switch"
			aria-checked={checked}
			aria-label={label}
			disabled={disabled}
			onClick={onToggle}
			class={cn(
				"relative h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors disabled:opacity-50",
				checked ? "bg-(--dominant-color)" : "bg-zinc-700",
			)}
		>
			<span
				class={cn(
					"absolute left-0.5 top-0.5 size-5 rounded-full transition-transform",
					// The knob has to contrast with the track, not with the page: the
					// binary pairs with the dominant colour on, but the off track is
					// dark, so it would swallow a black knob.
					checked
						? "translate-x-5 bg-(--binary-color)"
						: "translate-x-0 bg-white",
				)}
			/>
		</button>
	);
}

export default Switch;

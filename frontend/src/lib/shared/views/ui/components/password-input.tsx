import { Eye, EyeOff } from "lucide-preact";
import { useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

interface PasswordInputProps {
	id?: string;
	class?: string;
	placeholder?: string;
	value: string;
	onValue: (value: string) => void;
	onBlur?: () => void;
	label?: string;
}

export function PasswordInput({
	id,
	class: className,
	placeholder,
	value,
	onValue,
	onBlur,
	label,
}: PasswordInputProps) {
	const [visible, setVisible] = useState(false);

	return (
		<div class="relative">
			<input
				id={id}
				type={visible ? "text" : "password"}
				class={cn(className, "pr-11")}
				placeholder={placeholder}
				aria-label={label}
				value={value}
				onInput={(e) => onValue(e.currentTarget.value)}
				onBlur={onBlur}
			/>
			<button
				type="button"
				tabIndex={-1}
				aria-label={visible ? "Hide" : "Show"}
				onClick={() => setVisible((v) => !v)}
				class="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-zinc-500 transition-colors hover:text-zinc-200 cursor-pointer"
			>
				{visible ? <EyeOff size={16} /> : <Eye size={16} />}
			</button>
		</div>
	);
}

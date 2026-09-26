import { useSignal } from "@preact/signals";
import { Eye, EyeOff } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

interface PasswordInputProps {
	id?: string;
	class?: string;
	placeholder?: string;
	value: string;
	disabled?: boolean;
	onValue: (value: string) => void;
	onBlur?: () => void;
	label?: string;
}

export function PasswordInput({
	id,
	class: className,
	placeholder,
	value,
	disabled,
	onValue,
	onBlur,
	label,
}: PasswordInputProps) {
	const visible = useSignal(false);

	return (
		<div class="relative">
			<input
				id={id}
				type={visible.value ? "text" : "password"}
				class={cn(className, "pr-11")}
				placeholder={placeholder}
				aria-label={label}
				value={value}
				disabled={disabled}
				onInput={(e) => onValue(e.currentTarget.value)}
				onBlur={onBlur}
			/>
			<button
				type="button"
				tabIndex={-1}
				aria-label={visible.value ? t("password.hide") : t("password.show")}
				onClick={() => {
					visible.value = !visible.value;
				}}
				class="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-zinc-500 transition-colors hover:text-zinc-200 cursor-pointer"
			>
				{visible.value ? <EyeOff size={16} /> : <Eye size={16} />}
			</button>
		</div>
	);
}

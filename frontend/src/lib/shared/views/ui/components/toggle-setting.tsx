import { useSignal } from "@preact/signals";
import type { ComponentChildren } from "preact";
import { useEffect } from "preact/hooks";
import { Switch } from "@/lib/shared/views/ui/components/switch";

type Props = {
	icon: ComponentChildren;
	title: string;
	description: string;
	label: string;
	load: () => Promise<boolean>;
	save: (enabled: boolean) => Promise<unknown>;
	// changed runs once the value is stored, for what the switch turns on.
	changed?: (enabled: boolean) => void;
};

// ToggleSetting is one switch of the settings page: the value is read once, and
// a flip the backend refuses falls back to what it was.
export function ToggleSetting({
	icon,
	title,
	description,
	label,
	load,
	save,
	changed,
}: Props) {
	const enabled = useSignal(false);
	const ready = useSignal(false);

	useEffect(() => {
		let alive = true;
		const apply = (value: boolean) => {
			if (!alive) return;
			enabled.value = value;
			ready.value = true;
		};
		load().then(apply, () => apply(false));
		return () => {
			alive = false;
		};
	}, []);

	async function toggle() {
		if (!ready.value) return;
		const next = !enabled.value;
		enabled.value = next;
		try {
			await save(next);
			changed?.(next);
		} catch {
			enabled.value = !next;
		}
	}

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center justify-between gap-2">
				<div class="flex items-center gap-2">
					{icon}
					<h2 class="text-sm font-semibold text-zinc-200">{title}</h2>
				</div>
				{ready.value ? (
					<Switch
						checked={enabled.value}
						label={label}
						onToggle={() => void toggle()}
					/>
				) : (
					<div class="shimmer h-6 w-11 shrink-0 rounded-full" />
				)}
			</div>
			<p class="text-balance text-xs text-zinc-500">{description}</p>
		</section>
	);
}

export default ToggleSetting;

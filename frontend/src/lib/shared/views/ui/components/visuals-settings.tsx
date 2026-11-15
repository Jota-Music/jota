// Hallmark · component: per-field reset icon button · genre: atmospheric
// theme: project system (zinc surfaces, cover accent) · states: default ·
// hover · focus-visible · active · disabled (loading/error/success: a local
// reset cannot be in flight nor fail)
import { useSignal } from "@preact/signals";
import { Palette, RotateCcw } from "lucide-preact";
import { useEffect } from "preact/hooks";
import {
	backdrop,
	defaults,
	moteSize,
	setBackdrop,
	setMoteSize,
	setStrength,
	strength,
} from "@/lib/music/views/stores/visuals";
import { t } from "@/lib/shared/i18n";
import { Switch } from "@/lib/shared/views/ui/components/switch";

// A number, not a slider: this is a value the user sets once and then reads
// back, and a drag has no exact position to return to. It commits on blur, so
// the intermediate states of typing (an empty box, a lone minus) never reach
// the store and the field cannot fight the keyboard.
function Percent({
	label,
	value,
	min,
	max,
	defaultValue,
	disabled,
	onCommit,
	onReset,
}: {
	label: string;
	value: number;
	min: number;
	max: number;
	defaultValue: number;
	disabled: boolean;
	onCommit: (value: number) => void;
	onReset: () => void;
}) {
	const draft = useSignal(String(value));

	useEffect(() => {
		draft.value = String(value);
	}, [value]);

	function commit() {
		const parsed = Number(draft.value);
		const next = Number.isFinite(parsed)
			? Math.min(max, Math.max(min, Math.round(parsed)))
			: value;
		draft.value = String(next);
		onCommit(next);
	}

	return (
		<div class="flex flex-col gap-1.5">
			<span class="text-sm text-zinc-200">{label}</span>
			<span class="flex items-center gap-2">
				<input
					type="number"
					inputMode="numeric"
					min={min}
					max={max}
					aria-label={label}
					disabled={disabled}
					value={draft.value}
					onInput={(e) => (draft.value = e.currentTarget.value)}
					onBlur={commit}
					onKeyDown={(e) => {
						if (e.key === "Enter") e.currentTarget.blur();
					}}
					class="w-16 rounded-lg border border-zinc-800 bg-zinc-950 px-2 py-1.5 text-sm text-zinc-100 outline-none transition duration-150 hover:bg-zinc-900 focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-zinc-950"
				/>
				<span class="text-xs text-zinc-500">%</span>
				<button
					type="button"
					disabled={disabled || value === defaultValue}
					aria-label={`${label}: ${t("settings.visuals.reset")}`}
					title={t("settings.visuals.reset")}
					onClick={() => {
						draft.value = String(defaultValue);
						onReset();
					}}
					class="flex size-8 items-center justify-center rounded-lg border border-zinc-800 text-zinc-500 transition duration-150 hover:border-zinc-600 hover:bg-zinc-900 hover:text-zinc-200 active:translate-y-px focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:border-zinc-800 disabled:hover:bg-transparent disabled:hover:text-zinc-500 disabled:active:translate-y-0"
				>
					<RotateCcw size={13} />
				</button>
			</span>
		</div>
	);
}

export function VisualsSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Palette size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("settings.visuals.title")}
				</h2>
			</div>

			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<p class="text-sm text-zinc-200">
						{t("settings.visuals.background.label")}
					</p>
					<p class="text-balance text-xs text-zinc-500">
						{t("settings.visuals.background.description")}
					</p>
				</div>
				<Switch
					checked={backdrop.value}
					label={t("settings.visuals.background.label")}
					onToggle={() => setBackdrop(!backdrop.value)}
				/>
			</div>

			<div class="flex flex-col gap-4">
				<Percent
					label={t("settings.visuals.strength.label")}
					value={Math.round(strength.value * 100)}
					min={5}
					max={50}
					defaultValue={defaults.strength * 100}
					disabled={!backdrop.value}
					onCommit={(value) => setStrength(value / 100)}
					onReset={() => setStrength(defaults.strength)}
				/>
				<Percent
					label={t("settings.visuals.particleSize.label")}
					value={Math.round(moteSize.value * 100)}
					min={40}
					max={250}
					defaultValue={defaults.moteSize * 100}
					disabled={!backdrop.value}
					onCommit={(value) => setMoteSize(value / 100)}
					onReset={() => setMoteSize(defaults.moteSize)}
				/>
			</div>
		</section>
	);
}

export default VisualsSettings;

import { t } from "@/lib/shared/i18n";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export function YouTubeHint() {
	const hintSteps = [
		t("music.hint.step1"),
		t("music.hint.step2"),
		t("music.hint.step3"),
	];

	return (
		<div class="relative flex w-full max-w-sm flex-col items-center gap-5 px-6 py-8 text-center">
			<div class="pointer-events-none absolute top-0 size-32 rounded-full bg-red-600/20 blur-3xl" />

			<div class="relative flex size-12 items-center justify-center rounded-full bg-red-500/10 ring-1 ring-red-500/25 ring-inset">
				<YoutubeIcon class="size-6 text-red-500" />
			</div>

			<div class="relative flex flex-col gap-1">
				<p class="text-sm font-semibold text-zinc-100">
					{t("music.hint.title")}
				</p>
				<p class="text-xs text-zinc-500">{t("music.hint.body")}</p>
			</div>

			<ol class="relative flex w-full flex-col text-left">
				{hintSteps.map((step, index) => (
					<li key={step} class="relative flex gap-3 pb-4 last:pb-0">
						{index < hintSteps.length - 1 && (
							<span class="absolute left-3 top-7 h-[calc(100%-1.75rem)] w-px bg-zinc-800" />
						)}
						<span class="relative z-10 flex size-6 shrink-0 items-center justify-center rounded-full border border-zinc-700 bg-zinc-900 text-[11px] font-semibold text-zinc-300 tabular-nums">
							<span class="translate-y-[0.06em] leading-none">{index + 1}</span>
						</span>
						<span class="pt-0.5 text-xs leading-snug text-zinc-400">
							{step}
						</span>
					</li>
				))}
			</ol>
		</div>
	);
}

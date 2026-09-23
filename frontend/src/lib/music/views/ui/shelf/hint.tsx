import { Heart, Library } from "lucide-preact";
import type { ComponentChild } from "preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

interface HintProps {
	icon: ComponentChild;
	blob: string;
	badge: string;
	title: string;
	body: string;
	steps: string[];
}

export function Hint({ icon, blob, badge, title, body, steps }: HintProps) {
	return (
		<div class="relative flex w-full max-w-sm flex-col items-center gap-5 px-6 py-8 text-center">
			<div
				class={cn(
					"pointer-events-none absolute top-0 size-32 rounded-full blur-3xl",
					blob,
				)}
			/>

			<div
				class={cn(
					"relative flex size-12 items-center justify-center rounded-full ring-1 ring-inset",
					badge,
				)}
			>
				{icon}
			</div>

			<div class="relative flex flex-col gap-1">
				<p class="text-sm font-semibold text-zinc-100">{title}</p>
				<p class="text-xs text-zinc-500">{body}</p>
			</div>

			<ol class="relative flex w-full flex-col text-left">
				{steps.map((step, index) => (
					<li key={step} class="relative flex gap-3 pb-4 last:pb-0">
						{index < steps.length - 1 && (
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

export function YouTubeHint() {
	return (
		<Hint
			icon={<YoutubeIcon class="size-6 text-red-500" />}
			blob="bg-red-600/20"
			badge="bg-red-500/10 ring-red-500/25"
			title={t("music.hint.title")}
			body={t("music.hint.body")}
			steps={[
				t("music.hint.step1"),
				t("music.hint.step2"),
				t("music.hint.step3"),
			]}
		/>
	);
}

export function FollowingHint() {
	return (
		<Hint
			icon={<Heart class="size-6 text-fuchsia-500 fill-fuchsia-500/20" />}
			blob="bg-fuchsia-600/20"
			badge="bg-fuchsia-500/10 ring-fuchsia-500/25"
			title={t("music.following.hint.title")}
			body={t("music.following.hint.body")}
			steps={[
				t("music.following.hint.step1"),
				t("music.following.hint.step2"),
				t("music.following.hint.step3"),
			]}
		/>
	);
}

export function LocalHint() {
	return (
		<Hint
			icon={<Library class="size-6 text-emerald-400" />}
			blob="bg-emerald-500/20"
			badge="bg-emerald-500/10 ring-emerald-500/25"
			title={t("music.local.hint.title")}
			body={t("music.local.hint.body")}
			steps={[
				t("music.local.hint.step1"),
				t("music.local.hint.step2"),
				t("music.local.hint.step3"),
			]}
		/>
	);
}

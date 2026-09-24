import { Download, ExternalLink, Info, Loader2 } from "lucide-preact";
import { useEffect } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { open } from "@/lib/shared/utils/open";
import {
	checkUpdate,
	install,
	installing,
	percent,
	update,
	version,
} from "@/lib/update/views/stores/update";

export function AboutSettings() {
	useEffect(() => {
		void checkUpdate();
	}, []);

	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Info size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("update.about.title")}
				</h2>
			</div>

			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<div class="min-w-0">
					<p class="text-xs text-zinc-500">{t("update.about.version")}</p>
					<p class="truncate text-sm text-zinc-100">
						{version.value || t("update.about.dev")}
					</p>
				</div>
				<UpdateAction />
			</div>
		</section>
	);
}

function UpdateAction() {
	const info = update.value;

	if (!info?.available) {
		return null;
	}

	const p = installing.value ? percent() : null;

	return (
		<button
			type="button"
			disabled={installing.value}
			onClick={() => (info.installable ? void install() : void open(info.url))}
			class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg bg-(--dominant-color) px-3 py-2 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85 disabled:opacity-50"
		>
			{installing.value ? (
				<Loader2 size={12} class="animate-spin" />
			) : info.installable ? (
				<Download size={12} />
			) : (
				<ExternalLink size={12} />
			)}
			{installing.value
				? p == null
					? t("update.updating")
					: t("update.updatingPercent", { percent: p })
				: info.installable
					? t("update.about.updateTo", { version: info.latest })
					: t("update.about.download", { version: info.latest })}
		</button>
	);
}

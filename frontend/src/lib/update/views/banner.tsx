import {
	ArrowDownToLine,
	Download,
	ExternalLink,
	Loader2,
	X,
} from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { open } from "@/lib/shared/utils/open";
import {
	dismissed,
	install,
	installing,
	percent,
	update,
} from "@/lib/update/views/stores/update";

export function UpdateBanner() {
	const info = update.value;
	if (!info?.available || dismissed.value) return null;

	const p = installing.value ? percent() : null;

	return (
		<div class="relative flex items-center gap-3 border-b border-zinc-800 bg-zinc-900/60 px-4 py-2 text-sm">
			<ArrowDownToLine size={16} class="shrink-0 text-(--dominant-color)" />

			<p class="min-w-0 flex-1 truncate text-zinc-400">
				{t("update.available", { version: info.latest })}
			</p>

			{installing.value ? (
				<span class="flex shrink-0 items-center gap-1.5 rounded-full bg-zinc-800 px-3 py-1 text-xs font-semibold text-zinc-300">
					<Loader2 size={12} class="animate-spin" />
					{p == null
						? t("update.updating")
						: t("update.updatingPercent", { percent: p })}
				</span>
			) : (
				<button
					type="button"
					onClick={() =>
						info.installable ? void install() : void open(info.url)
					}
					class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-full bg-(--dominant-color) px-3 py-1 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85"
				>
					{info.installable ? (
						<Download size={12} />
					) : (
						<ExternalLink size={12} />
					)}
					{info.installable ? t("update.update") : t("update.viewRelease")}
				</button>
			)}

			<button
				type="button"
				onClick={() => (dismissed.value = true)}
				aria-label={t("common.dismiss")}
				class="shrink-0 cursor-pointer text-zinc-500 transition-colors hover:text-zinc-200"
			>
				<X size={14} />
			</button>

			{p != null && (
				<div class="absolute inset-x-0 bottom-0 h-0.5 bg-zinc-800">
					<div
						class="h-full origin-left bg-(--dominant-color) transition-[scale] duration-150 ease-out"
						style={{ scale: p / 100 }}
					/>
				</div>
			)}
		</div>
	);
}

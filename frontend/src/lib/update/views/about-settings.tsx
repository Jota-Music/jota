import { Download, ExternalLink, Info, Loader2 } from "lucide-preact";
import { useEffect } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { browse } from "@/lib/shared/utils/open";
import {
	checkUpdate,
	install,
	installing,
	percent,
	update,
	version,
} from "@/lib/update/views/stores/update";

const RELEASES_URL = "https://github.com/Jota-Music/jota/releases";

function Github({ size = 16 }: { size?: number }) {
	return (
		<svg
			width={size}
			height={size}
			viewBox="0 0 24 24"
			fill="currentColor"
			aria-hidden="true"
		>
			<path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
		</svg>
	);
}

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
				<div class="flex shrink-0 items-center gap-2">
					<button
						type="button"
						title={t("update.viewRelease")}
						aria-label={t("update.viewRelease")}
						onClick={() => browse(RELEASES_URL)}
						class="shrink-0 cursor-pointer text-zinc-500 transition-colors hover:text-zinc-200"
					>
						<Github size={16} />
					</button>
					<UpdateAction />
				</div>
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
			onClick={() => (info.installable ? void install() : browse(info.url))}
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

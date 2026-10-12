import { Link } from "wouter-preact";
import { t } from "@/lib/shared/i18n";

export function NotFoundPage() {
	return (
		<div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 text-sm">
			<p className="text-zinc-400">{t("pages.notFound.title")}</p>
			<Link
				href="/"
				className="cursor-pointer rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 transition-colors hover:bg-zinc-800"
			>
				{t("pages.notFound.home")}
			</Link>
		</div>
	);
}

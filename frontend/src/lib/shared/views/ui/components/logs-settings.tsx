import { SaveLogs } from "@bindings/app";
import { FileDown } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { addError } from "@/lib/shared/views/stores/errors";

async function save(): Promise<void> {
	try {
		await SaveLogs();
	} catch (err) {
		addError(err, "logs");
	}
}

export function LogsSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<FileDown size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("settings.logs.title")}
				</h2>
			</div>

			<div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-950 p-3 ring-1 ring-zinc-800">
				<p class="min-w-0 text-xs text-zinc-500">
					{t("settings.logs.description")}
				</p>
				<button
					type="button"
					onClick={() => void save()}
					class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-lg bg-(--dominant-color) px-3 py-2 text-xs font-semibold text-(--binary-color) transition-opacity hover:opacity-85"
				>
					<FileDown size={12} />
					{t("settings.logs.download")}
				</button>
			</div>
		</section>
	);
}

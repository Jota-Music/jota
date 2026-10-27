import { ArchiveRestore } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { Autosave } from "@/lib/shared/views/ui/components/autosave";
import { BackupFile } from "@/lib/shared/views/ui/components/backup-file";

export function BackupsSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<ArchiveRestore size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("settings.backups.title")}
				</h2>
			</div>

			<BackupFile />
			<div class="border-t border-zinc-800" />
			<Autosave />
		</section>
	);
}

import { Loader } from "lucide-preact";
import { t } from "@/lib/shared/i18n";

export function Loading() {
	return (
		<div
			role="status"
			class="flex items-center justify-center gap-2 py-10 text-sm text-zinc-500"
		>
			<Loader size={20} class="animate-spin" />
			{t("common.loading")}
		</div>
	);
}

import { ArrowUpDown } from "lucide-preact";
import { t } from "@/lib/shared/i18n";

export type OrderType = "added" | "reverse" | "duration-asc" | "duration-desc";

export function OrderSelect({
	value,
	onChange,
}: {
	value: OrderType;
	onChange: (order: OrderType) => void;
}) {
	return (
		<div class="relative">
			<ArrowUpDown
				size={16}
				class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
			/>
			<select
				value={value}
				onChange={(e) =>
					onChange((e.target as HTMLSelectElement).value as OrderType)
				}
				aria-label={t("music.playlist.orderBy")}
				class="h-10 w-max cursor-pointer appearance-none rounded-md border border-zinc-800 bg-zinc-950 pl-9 pr-3 text-sm text-white outline-none focus:border-zinc-600"
			>
				<option value="added">{t("music.playlist.order.added")}</option>
				<option value="reverse">{t("music.playlist.order.reverse")}</option>
				<option value="duration-asc">
					{t("music.playlist.order.durationAsc")}
				</option>
				<option value="duration-desc">
					{t("music.playlist.order.durationDesc")}
				</option>
			</select>
		</div>
	);
}

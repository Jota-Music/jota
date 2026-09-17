import { ChevronDown, Languages } from "lucide-preact";
import { available, locale, setLocale, t } from "@/lib/shared/i18n";

function name(code: string): string {
	try {
		return new Intl.DisplayNames([code], { type: "language" }).of(code) ?? code;
	} catch {
		return code;
	}
}

export function LanguageSettings() {
	return (
		<section class="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-4">
			<div class="flex items-center gap-2">
				<Languages size={18} class="text-zinc-400" />
				<h2 class="text-sm font-semibold text-zinc-200">
					{t("settings.language.title")}
				</h2>
			</div>
			<p class="text-xs text-zinc-500">{t("settings.language.description")}</p>
			<div class="relative">
				<label for="settings-language" class="sr-only">
					{t("settings.language.label")}
				</label>
				<select
					id="settings-language"
					value={locale.value}
					onChange={(e) => setLocale((e.target as HTMLSelectElement).value)}
					class="w-full cursor-pointer appearance-none rounded-lg border border-zinc-800 bg-zinc-950 p-3 pr-10 text-sm text-zinc-100 outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-all"
				>
					{available.map((code) => (
						<option key={code} value={code} class="bg-zinc-950">
							{name(code)}
						</option>
					))}
				</select>
				<ChevronDown class="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
			</div>
		</section>
	);
}

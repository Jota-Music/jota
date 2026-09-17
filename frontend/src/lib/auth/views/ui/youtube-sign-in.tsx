import { useState } from "preact/hooks";
import {
	loginYouTubeWithBrowser,
	saveYouTubeCookies,
	youtubeBrowserLoginSupported,
} from "@/lib/auth/views/stores/youtube";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";

function message(err: unknown): string {
	return translateError(err);
}

export function YouTubeSignIn() {
	const [cookies, setCookies] = useState("");
	const [busy, setBusy] = useState(false);
	const [browserBusy, setBrowserBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function signInWithBrowser() {
		setBrowserBusy(true);
		setError(null);
		try {
			await loginYouTubeWithBrowser();
		} catch (err) {
			setError(message(err));
		} finally {
			setBrowserBusy(false);
		}
	}

	async function save(value: string) {
		setBusy(true);
		setError(null);
		try {
			await saveYouTubeCookies(value.trim());
			setCookies("");
		} catch (err) {
			setError(message(err));
		} finally {
			setBusy(false);
		}
	}

	async function importFile(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) await save(await file.text());
		input.value = "";
	}

	return (
		<div class="space-y-3">
			<div class="rounded-lg border border-amber-900/60 bg-amber-950/30 p-3 text-xs text-amber-200">
				<strong class="font-semibold">{t("auth.youtube.riskLabel")}</strong>{" "}
				{t("auth.youtube.settingsRisk")}
			</div>

			{youtubeBrowserLoginSupported.value ? (
				<>
					<button
						type="button"
						onClick={() => void signInWithBrowser()}
						disabled={browserBusy}
						class="w-full rounded-lg bg-zinc-800 px-3 py-2 text-xs font-semibold text-zinc-100 hover:bg-zinc-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
					>
						{browserBusy
							? t("auth.youtube.waiting")
							: t("auth.youtube.signInBrowser")}
					</button>

					<div class="flex items-center gap-2 text-[11px] text-zinc-600">
						<span class="h-px flex-1 bg-zinc-800" />
						{t("auth.youtube.orPaste")}
						<span class="h-px flex-1 bg-zinc-800" />
					</div>
				</>
			) : (
				<p class="text-xs leading-relaxed text-zinc-500">
					{t("auth.youtube.unavailableShort")}
				</p>
			)}

			<form
				onSubmit={(e) => {
					e.preventDefault();
					void save(cookies);
				}}
				class="space-y-3"
			>
				<p class="text-xs leading-relaxed text-zinc-500">
					{t("auth.youtube.pasteHint")}
				</p>

				<textarea
					value={cookies}
					onInput={(e) => setCookies((e.target as HTMLTextAreaElement).value)}
					rows={3}
					spellcheck={false}
					placeholder={t("auth.youtube.pastePlaceholder")}
					class="w-full resize-none rounded-lg bg-zinc-950 p-3 text-xs text-zinc-200 ring-1 ring-zinc-800 outline-none focus:ring-zinc-600"
				/>

				<div class="flex items-center gap-2">
					<button
						type="submit"
						disabled={busy || !cookies.trim()}
						class="rounded-lg bg-zinc-800 px-3 py-2 text-xs font-semibold text-zinc-100 hover:bg-zinc-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
					>
						{busy ? t("common.saving") : t("common.save")}
					</button>
					<label class="cursor-pointer rounded-lg px-3 py-2 text-xs font-semibold text-zinc-400 hover:text-zinc-200 transition-colors">
						{t("auth.youtube.importFile")}
						<input
							type="file"
							accept=".txt,text/plain"
							onChange={importFile}
							disabled={busy}
							class="hidden"
						/>
					</label>
				</div>
			</form>

			{error && <p class="text-xs text-red-400">{error}</p>}
		</div>
	);
}

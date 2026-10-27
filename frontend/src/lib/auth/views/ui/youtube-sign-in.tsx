import { useSignal } from "@preact/signals";
import { useQueryClient } from "@tanstack/preact-query";
import { ChevronDown, ChevronRight, Loader, ShieldAlert } from "lucide-preact";
import { useEffect } from "preact/hooks";
import {
	browserAccounts,
	browsers,
	loginYouTubeWithBrowser,
	saveYouTubeCookies,
	signInFromBrowser,
	youtubeBrowserLoginSupported,
} from "@/lib/auth/views/stores/youtube";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";

function message(err: unknown): string {
	return translateError(err);
}

// BrowserImport adopts the session a browser on this machine already holds, the
// one action that needs no window and no file. It is also the way out of a
// partial session, so the signed-in panel offers it too.
export function BrowserImport() {
	const importBusy = useSignal(false);
	const installed = useSignal<string[]>([]);
	const accounts = useSignal<Record<string, string>>({});
	const browser = useSignal("");
	const error = useSignal<string | null>(null);
	const queryClient = useQueryClient();

	useEffect(() => {
		void browsers().then((found) => {
			installed.value = found;
			browser.value = found[0] ?? "";
			// The accounts name themselves after a round trip, so they land a
			// moment later and relabel the options.
			void browserAccounts().then((named) => {
				accounts.value = named;
			});
		});
	}, []);

	async function run() {
		importBusy.value = true;
		error.value = null;
		try {
			await signInFromBrowser(browser.value);
			await queryClient.invalidateQueries({
				queryKey: ["user-playlists", "youtube"],
			});
		} catch (err) {
			error.value = message(err);
		} finally {
			importBusy.value = false;
		}
	}

	if (installed.value.length === 0) return null;

	const label = accounts.value[browser.value]
		? `${browser.value} — ${accounts.value[browser.value]}`
		: browser.value;

	return (
		<div class="space-y-3">
			{installed.value.length > 1 && (
				<div class="relative">
					<label for="youtube-browser" class="sr-only">
						{t("auth.youtube.browserImportPick")}
					</label>
					<select
						id="youtube-browser"
						value={browser.value}
						onChange={(e) => {
							browser.value = (e.target as HTMLSelectElement).value;
						}}
						class="w-full cursor-pointer appearance-none rounded-lg border border-zinc-800 bg-zinc-950 p-3 pr-10 text-sm text-zinc-100 outline-none transition-all focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
					>
						{installed.value.map((name) => (
							<option key={name} value={name} class="bg-zinc-950">
								{accounts.value[name]
									? `${name} — ${accounts.value[name]}`
									: name}
							</option>
						))}
					</select>
					<ChevronDown class="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
				</div>
			)}

			{installed.value.length === 1 && (
				<p class="text-xs text-zinc-500">{label}</p>
			)}

			<button
				type="button"
				onClick={() => void run()}
				disabled={importBusy.value || browser.value === ""}
				class="flex w-full cursor-pointer items-center justify-center gap-2 rounded-lg bg-zinc-800 px-5 py-2.5 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{importBusy.value && <Loader size={14} class="animate-spin" />}
				{importBusy.value
					? t("auth.youtube.browserImportBusy", { browser: browser.value })
					: t("auth.youtube.browserImport", { browser: browser.value })}
			</button>

			<p class="text-xs text-zinc-500">
				{t("auth.youtube.browserImportHint", { browser: browser.value })}
			</p>

			{error.value && <p class="text-xs text-red-400">{error.value}</p>}
		</div>
	);
}

export function YouTubeSignIn() {
	const cookies = useSignal("");
	const busy = useSignal(false);
	const browserBusy = useSignal(false);
	const error = useSignal<string | null>(null);

	async function signInWithBrowser() {
		browserBusy.value = true;
		error.value = null;
		try {
			await loginYouTubeWithBrowser();
		} catch (err) {
			error.value = message(err);
		} finally {
			browserBusy.value = false;
		}
	}

	async function save(value: string) {
		busy.value = true;
		error.value = null;
		try {
			await saveYouTubeCookies(value.trim());
			cookies.value = "";
		} catch (err) {
			error.value = message(err);
		} finally {
			busy.value = false;
		}
	}

	async function importFile(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) await save(await file.text());
		input.value = "";
	}

	return (
		<div class="space-y-4">
			<BrowserImport />

			{youtubeBrowserLoginSupported.value ? (
				<button
					type="button"
					onClick={() => void signInWithBrowser()}
					disabled={browserBusy.value}
					class="flex w-full cursor-pointer items-center justify-center gap-2 rounded-lg border border-zinc-800 px-3 py-2.5 text-xs font-semibold text-zinc-300 transition-colors hover:border-zinc-700 hover:text-zinc-100 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{browserBusy.value && <Loader size={14} class="animate-spin" />}
					{browserBusy.value
						? t("auth.youtube.waiting")
						: t("auth.youtube.signInWindow")}
				</button>
			) : (
				<p class="text-xs leading-relaxed text-zinc-500">
					{t("auth.youtube.unavailableShort")}
				</p>
			)}

			<details class="group rounded-xl border border-zinc-800 bg-zinc-950/60">
				<summary class="flex cursor-pointer list-none items-center gap-2 p-3 text-xs font-semibold text-zinc-400 transition-colors hover:text-zinc-200 [&::-webkit-details-marker]:hidden">
					<ChevronRight class="size-3.5 shrink-0 transition-transform group-open:rotate-90" />
					{t("auth.youtube.cookiesManual")}
				</summary>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						void save(cookies.value);
					}}
					class="space-y-3 border-t border-zinc-800 p-3"
				>
					<p class="text-xs leading-relaxed text-zinc-500">
						{t("auth.youtube.pasteHint")}
					</p>

					<textarea
						value={cookies.value}
						onInput={(e) => {
							cookies.value = (e.target as HTMLTextAreaElement).value;
						}}
						rows={3}
						spellcheck={false}
						placeholder={t("auth.youtube.pastePlaceholder")}
						class="w-full resize-none rounded-lg border border-zinc-800 bg-zinc-950 p-3 text-xs text-zinc-200 outline-none transition-all focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600"
					/>

					<div class="flex items-center gap-2">
						<button
							type="submit"
							disabled={busy.value || !cookies.value.trim()}
							class="cursor-pointer rounded-lg bg-zinc-800 px-3 py-2 text-xs font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 disabled:cursor-not-allowed disabled:opacity-50"
						>
							{busy.value ? t("common.saving") : t("common.save")}
						</button>
						<label class="cursor-pointer rounded-lg px-3 py-2 text-xs font-semibold text-zinc-400 transition-colors hover:text-zinc-200">
							{t("auth.youtube.importFile")}
							<input
								type="file"
								accept=".txt,text/plain"
								onChange={importFile}
								disabled={busy.value}
								class="hidden"
							/>
						</label>
					</div>
				</form>
			</details>

			<p class="flex gap-2 text-xs leading-relaxed text-amber-300/80">
				<ShieldAlert size={14} class="mt-0.5 shrink-0" />
				{t("auth.youtube.settingsRisk")}
			</p>

			{error.value && <p class="text-xs text-red-400">{error.value}</p>}
		</div>
	);
}

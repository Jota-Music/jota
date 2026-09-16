import { useState } from "preact/hooks";
import { saveYouTubeCookies } from "@/lib/auth/views/stores/youtube";

export function YouTubeSignIn() {
	const [cookies, setCookies] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function save(value: string) {
		setBusy(true);
		setError(null);
		try {
			await saveYouTubeCookies(value.trim());
			setCookies("");
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
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
		<form
			onSubmit={(e) => {
				e.preventDefault();
				void save(cookies);
			}}
			class="space-y-3"
		>
			<div class="rounded-lg border border-amber-900/60 bg-amber-950/30 p-3 text-xs text-amber-200">
				<strong class="font-semibold">Risk.</strong> Signing in with a Google
				account and using it to play audio can get that account banned by
				Google. Use an account you can afford to lose, and prefer a secondary
				one.
			</div>

			<p class="text-xs leading-relaxed text-zinc-500">
				On a signed-in youtube.com tab, run{" "}
				<code class="rounded bg-zinc-800 px-1 py-0.5 text-zinc-300">
					copy(document.cookie)
				</code>{" "}
				in the console and paste it below, or import a{" "}
				<code class="rounded bg-zinc-800 px-1 py-0.5 text-zinc-300">
					cookies.txt
				</code>{" "}
				export.
			</p>

			<textarea
				value={cookies}
				onInput={(e) => setCookies((e.target as HTMLTextAreaElement).value)}
				rows={3}
				spellcheck={false}
				placeholder="Paste cookies (must include SAPISID) or a cookies.txt export"
				class="w-full resize-none rounded-lg bg-zinc-950 p-3 text-xs text-zinc-200 ring-1 ring-zinc-800 outline-none focus:ring-zinc-600"
			/>
			{error && <p class="text-xs text-red-400">{error}</p>}

			<div class="flex items-center gap-2">
				<button
					type="submit"
					disabled={busy || !cookies.trim()}
					class="rounded-lg bg-zinc-800 px-3 py-2 text-xs font-semibold text-zinc-100 hover:bg-zinc-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
				>
					{busy ? "Saving…" : "Sign in"}
				</button>
				<label class="cursor-pointer rounded-lg px-3 py-2 text-xs font-semibold text-zinc-400 hover:text-zinc-200 transition-colors">
					Import cookies.txt
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
	);
}

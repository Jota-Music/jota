import { useState } from "preact/hooks";
import { saveYouTubeCookies } from "@/lib/auth/views/stores/youtube";

export function YouTubeSignIn() {
	const [cookies, setCookies] = useState("");
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function submit(event: Event) {
		event.preventDefault();
		setBusy(true);
		setError(null);
		try {
			await saveYouTubeCookies(cookies.trim());
			setCookies("");
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
		} finally {
			setBusy(false);
		}
	}

	return (
		<form onSubmit={submit} class="space-y-3">
			<div class="rounded-lg border border-amber-900/60 bg-amber-950/30 p-3 text-xs text-amber-200">
				<strong class="font-semibold">Risk.</strong> Signing in with a Google
				account and using it to play audio can get that account banned by
				Google. Use an account you can afford to lose, and prefer a secondary
				one.
			</div>
			<textarea
				value={cookies}
				onInput={(e) => setCookies((e.target as HTMLTextAreaElement).value)}
				rows={3}
				spellcheck={false}
				placeholder="Paste your youtube.com cookies (must include SAPISID)"
				class="w-full resize-none rounded-lg bg-zinc-950 p-3 text-xs text-zinc-200 ring-1 ring-zinc-800 outline-none focus:ring-zinc-600"
			/>
			{error && <p class="text-xs text-red-400">{error}</p>}
			<button
				type="submit"
				disabled={busy || !cookies.trim()}
				class="rounded-lg bg-zinc-800 px-3 py-2 text-xs font-semibold text-zinc-100 hover:bg-zinc-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
			>
				{busy ? "Saving…" : "Sign in"}
			</button>
		</form>
	);
}

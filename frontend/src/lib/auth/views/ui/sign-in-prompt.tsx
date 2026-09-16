import { Loader2, LogIn, X } from "lucide-preact";
import { useState } from "preact/hooks";
import {
	dismissYouTubeSignInSuggestion,
	loginYouTubeWithBrowser,
	youtubeSignInSuggested,
} from "@/lib/auth/views/stores/youtube";

export function SignInPrompt() {
	const [busy, setBusy] = useState(false);

	if (!youtubeSignInSuggested.value) return null;

	async function signIn() {
		setBusy(true);
		try {
			await loginYouTubeWithBrowser();
		} finally {
			setBusy(false);
		}
	}

	return (
		<div class="relative flex items-center gap-3 border-b border-amber-900/40 bg-amber-950/40 px-4 py-2 text-sm">
			<LogIn size={16} class="shrink-0 text-amber-300" />
			<p class="min-w-0 flex-1 truncate text-amber-200">
				This track needs a YouTube sign-in — it is age-restricted or YouTube
				asked to confirm you are not a bot.
			</p>
			<button
				type="button"
				onClick={() => void signIn()}
				disabled={busy}
				class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-full bg-amber-500 px-3 py-1 text-xs font-semibold text-amber-950 transition-opacity hover:opacity-85 disabled:opacity-60"
			>
				{busy ? <Loader2 size={12} class="animate-spin" /> : null}
				{busy ? "Waiting…" : "Sign in"}
			</button>
			<button
				type="button"
				onClick={() => dismissYouTubeSignInSuggestion()}
				aria-label="Dismiss"
				class="shrink-0 cursor-pointer text-amber-400/70 transition-colors hover:text-amber-200"
			>
				<X size={14} />
			</button>
		</div>
	);
}

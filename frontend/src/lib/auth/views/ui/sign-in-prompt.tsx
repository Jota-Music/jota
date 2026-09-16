import { useState } from "preact/hooks";
import {
	dismissYouTubeSignInSuggestion,
	loginYouTubeWithBrowser,
	youtubeSignInSuggested,
} from "@/lib/auth/views/stores/youtube";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export function SignInPrompt() {
	const [busy, setBusy] = useState(false);

	async function signIn() {
		setBusy(true);
		try {
			await loginYouTubeWithBrowser();
		} finally {
			setBusy(false);
		}
	}

	return (
		<Modal
			open={youtubeSignInSuggested.value}
			close={() => dismissYouTubeSignInSuggestion()}
			labelledBy="youtube-signin-title"
		>
			<div class="flex flex-col items-center gap-4 p-6 text-center">
				<YoutubeIcon width={36} height={36} class="text-[#FF0000]" />
				<div class="space-y-1">
					<h2 id="youtube-signin-title" class="text-lg font-bold text-zinc-100">
						Sign in to YouTube
					</h2>
					<p class="text-sm text-zinc-400">
						This track needs a signed-in YouTube account — it is age-restricted
						or YouTube asked to confirm you are not a bot.
					</p>
					<p class="text-xs text-amber-300/80">
						Risk: using an account this way can get it banned by Google. Prefer
						a secondary account.
					</p>
				</div>

				<div class="flex items-center gap-2">
					<button
						type="button"
						disabled={busy}
						onClick={() => void signIn()}
						class="rounded-full bg-[#FF0000] px-5 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-85 cursor-pointer disabled:opacity-50"
					>
						{busy ? "Waiting for sign-in…" : "Sign in"}
					</button>
					<button
						type="button"
						disabled={busy}
						onClick={() => dismissYouTubeSignInSuggestion()}
						class="rounded-full px-5 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-200 cursor-pointer disabled:opacity-50"
					>
						Not now
					</button>
				</div>
			</div>
		</Modal>
	);
}

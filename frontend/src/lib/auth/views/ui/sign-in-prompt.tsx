import { useState } from "preact/hooks";
import {
	dismissYouTubeSignInSuggestion,
	loginYouTubeWithBrowser,
	youtubeBrowserLoginSupported,
	youtubeSignInSuggested,
} from "@/lib/auth/views/stores/youtube";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";
import { Modal, ModalHeader } from "@/lib/shared/views/ui/components/modal";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export function SignInPrompt() {
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function signIn() {
		setBusy(true);
		setError(null);
		try {
			await loginYouTubeWithBrowser();
		} catch (err) {
			setError(translateError(err));
		} finally {
			setBusy(false);
		}
	}

	return (
		<Modal
			open={youtubeSignInSuggested.value}
			close={() => dismissYouTubeSignInSuggestion()}
			labelledBy="youtube-signin-title"
			hideClose
		>
			<ModalHeader
				close={() => dismissYouTubeSignInSuggestion()}
				bordered={false}
			/>
			<div class="flex flex-col items-center gap-4 px-6 pb-6 text-center">
				<YoutubeIcon width={36} height={36} class="text-[#FF0000]" />
				<div class="space-y-1">
					<h2 id="youtube-signin-title" class="text-lg font-bold text-zinc-100">
						{t("auth.youtube.signInTitle")}
					</h2>
					<p class="text-sm text-zinc-400">{t("auth.youtube.signInBody")}</p>
					<p class="text-xs text-amber-300/80">
						{t("auth.youtube.signInRisk")}
					</p>
				</div>

				{youtubeBrowserLoginSupported.value ? (
					<div class="flex items-center gap-2">
						<button
							type="button"
							disabled={busy}
							onClick={() => void signIn()}
							class="rounded-full bg-[#FF0000] px-5 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-85 cursor-pointer disabled:opacity-50"
						>
							{busy ? t("auth.youtube.waiting") : t("auth.youtube.signIn")}
						</button>
						<button
							type="button"
							disabled={busy}
							onClick={() => dismissYouTubeSignInSuggestion()}
							class="rounded-full px-5 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-200 cursor-pointer disabled:opacity-50"
						>
							{t("auth.notNow")}
						</button>
					</div>
				) : (
					<div class="space-y-3">
						<p class="text-xs text-zinc-500">{t("auth.youtube.unavailable")}</p>
						<button
							type="button"
							onClick={() => dismissYouTubeSignInSuggestion()}
							class="rounded-full bg-zinc-800 px-5 py-2 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 cursor-pointer"
						>
							OK
						</button>
					</div>
				)}

				{error && (
					<p class="text-xs text-red-400">
						{t("auth.youtube.errorHint", { error })}
					</p>
				)}
			</div>
		</Modal>
	);
}

import { useSignal } from "@preact/signals";
import { useRef } from "preact/hooks";
import {
	dismissYouTubeSignInSuggestion,
	loginYouTubeWithBrowser,
	youtubeBrowserLoginSupported,
	youtubeSignInSuggested,
} from "@/lib/auth/views/stores/youtube";
import { t } from "@/lib/shared/i18n";
import { translateError } from "@/lib/shared/i18n/errors";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import YoutubeIcon from "@/lib/shared/views/ui/icons/youtube";

export function SignInPrompt() {
	const busy = useSignal(false);
	const error = useSignal<string | null>(null);
	const listRef = useRef<HTMLDivElement>(null);

	async function signIn() {
		busy.value = true;
		error.value = null;
		try {
			await loginYouTubeWithBrowser();
		} catch (err) {
			error.value = translateError(err);
		} finally {
			busy.value = false;
		}
	}

	return (
		<Modal
			open={youtubeSignInSuggested.value}
			close={() => dismissYouTubeSignInSuggestion()}
			labelledBy="youtube-signin-title"
		>
			<div class="relative flex min-h-0 flex-1 flex-col">
				<div
					ref={listRef}
					class="flex min-h-0 flex-1 flex-col items-center gap-4 overflow-y-auto px-6 pt-6 pb-6 text-center"
				>
					<YoutubeIcon width={36} height={36} class="shrink-0 text-[#FF0000]" />
					<div class="shrink-0 space-y-1">
						<h2
							id="youtube-signin-title"
							class="text-lg font-bold text-zinc-100"
						>
							{t("auth.youtube.signInTitle")}
						</h2>
						<p class="text-balance text-sm text-zinc-400">
							{t("auth.youtube.signInBody")}
						</p>
						<p class="text-balance text-xs text-amber-300/80">
							{t("auth.youtube.signInRisk")}
						</p>
					</div>

					{youtubeBrowserLoginSupported.value ? (
						<div class="flex shrink-0 items-center gap-2">
							<button
								type="button"
								disabled={busy.value}
								onClick={() => void signIn()}
								class="rounded-full bg-[#FF0000] px-5 py-2 text-sm font-semibold text-white transition-opacity hover:opacity-85 cursor-pointer disabled:opacity-50"
							>
								{busy.value
									? t("auth.youtube.waiting")
									: t("auth.youtube.signIn")}
							</button>
							<button
								type="button"
								disabled={busy.value}
								onClick={() => dismissYouTubeSignInSuggestion()}
								class="rounded-full px-5 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-200 cursor-pointer disabled:opacity-50"
							>
								{t("auth.notNow")}
							</button>
						</div>
					) : (
						<div class="shrink-0 space-y-3">
							<p class="text-balance text-xs text-zinc-500">
								{t("auth.youtube.unavailable")}
							</p>
							<button
								type="button"
								onClick={() => dismissYouTubeSignInSuggestion()}
								class="rounded-full bg-zinc-800 px-5 py-2 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 cursor-pointer"
							>
								OK
							</button>
						</div>
					)}

					{error.value && (
						<p class="shrink-0 text-balance text-xs text-red-400">
							{t("auth.youtube.errorHint", { error: error.value })}
						</p>
					)}
				</div>
				<Scrollbar target={listRef} gutter={24} />
			</div>
		</Modal>
	);
}

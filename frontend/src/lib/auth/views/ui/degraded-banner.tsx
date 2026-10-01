import { useSignal } from "@preact/signals";
import { Loader2, WifiOff } from "lucide-preact";
import { retrySpotify, spotifyDegraded } from "@/lib/auth/views/stores/session";
import { t } from "@/lib/shared/i18n";

// Not dismissible: this is a standing state, not a one-off notice. Hiding it
// would leave search and radio failing with no visible explanation.
export function DegradedBanner() {
	const retrying = useSignal(false);
	if (!spotifyDegraded.value) return null;

	async function retry() {
		retrying.value = true;
		try {
			await retrySpotify();
		} finally {
			retrying.value = false;
		}
	}

	return (
		<div class="relative flex items-center gap-3 border-b border-zinc-800 bg-zinc-900/60 px-4 py-2 text-sm">
			<WifiOff size={16} class="shrink-0 text-amber-500/80" />

			<p class="min-w-0 flex-1 truncate text-zinc-400">
				{t("auth.spotify.offline")}
			</p>

			<button
				type="button"
				disabled={retrying.value}
				onClick={() => void retry()}
				class="flex shrink-0 cursor-pointer items-center gap-1.5 rounded-full bg-zinc-800 px-3 py-1 text-xs font-semibold text-zinc-200 transition-colors hover:bg-zinc-700 disabled:opacity-50"
			>
				{retrying.value ? (
					<Loader2 size={12} class="animate-spin" />
				) : (
					<WifiOff size={12} />
				)}
				{t("auth.spotify.retry")}
			</button>
		</div>
	);
}

import { Check } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import {
	loginSpotifyAndWait,
	spotifyConnected,
	spotifyReady,
} from "@/lib/auth/views/stores/session";
import {
	dismissWelcome,
	welcomeDismissed,
	welcomeOpen,
} from "@/lib/auth/views/stores/welcome";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { Modal } from "@/lib/shared/views/ui/components/modal";
import { SpotifyIcon } from "@/lib/shared/views/ui/icons/spotify";

export function Welcome() {
	const [busy, setBusy] = useState(false);
	const [mute, setMute] = useState(false);
	const ready = spotifyReady.value;
	const connected = spotifyConnected.value;

	useEffect(() => {
		if (ready && !connected && !welcomeDismissed()) {
			welcomeOpen.value = true;
		}
	}, [ready, connected]);

	async function connect() {
		setBusy(true);
		const ok = await loginSpotifyAndWait();
		setBusy(false);
		if (ok) dismissWelcome(mute);
	}

	return (
		<Modal
			open={welcomeOpen.value}
			close={() => dismissWelcome(mute)}
			labelledBy="welcome-title"
		>
			<div class="flex flex-col items-center gap-4 p-6 text-center">
				<SpotifyIcon size={32} class="text-green-500" />
				<div class="space-y-1">
					<h2 id="welcome-title" class="text-lg font-bold text-zinc-100">
						{t("auth.welcome.title")}
					</h2>
					<p class="text-sm text-zinc-400">{t("auth.welcome.body")}</p>
				</div>

				<div class="flex items-center gap-2">
					<button
						type="button"
						disabled={busy}
						onClick={() => void connect()}
						class="rounded-full bg-zinc-800 px-5 py-2 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 cursor-pointer disabled:opacity-50"
					>
						{busy ? t("auth.welcome.connecting") : t("auth.welcome.connect")}
					</button>
					<button
						type="button"
						disabled={busy}
						onClick={() => dismissWelcome(mute)}
						class="rounded-full px-5 py-2 text-sm font-medium text-zinc-400 transition-colors hover:bg-zinc-900 hover:text-zinc-200 cursor-pointer disabled:opacity-50"
					>
						{t("auth.notNow")}
					</button>
				</div>

				<label class="flex cursor-pointer select-none items-center gap-2 text-xs text-zinc-500 transition-colors hover:text-zinc-300">
					<input
						type="checkbox"
						checked={mute}
						onChange={(e) => setMute(e.currentTarget.checked)}
						class="peer sr-only"
					/>
					<span
						class={cn(
							"flex size-4 items-center justify-center rounded border transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-green-500/40",
							mute
								? "border-green-500 bg-green-500 text-zinc-950"
								: "border-zinc-700 bg-zinc-900 text-transparent",
						)}
					>
						<Check size={12} strokeWidth={3} />
					</span>
					{t("auth.welcome.dontShow")}
				</label>
			</div>
		</Modal>
	);
}

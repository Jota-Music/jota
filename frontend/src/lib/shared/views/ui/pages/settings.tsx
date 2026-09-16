import { useRef } from "preact/hooks";
import { AccountSettings } from "@/lib/auth/views/ui/account-settings";
import { open } from "@/lib/shared/utils/open";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";
import { AppShell } from "@/lib/shared/views/ui/layouts/app-shell";
import { RelaySettings } from "@/lib/sync/views/ui/relay-settings";
import { AboutSettings } from "@/lib/update/views/about-settings";

const GITHUB_URL = "https://github.com/salvadorsru";

function openDeveloper(): void {
	void open(GITHUB_URL);
}

function SettingsPage() {
	const listRef = useRef<HTMLDivElement>(null);

	return (
		<AppShell rootClass="pb-6">
			<div class="relative min-h-0 flex-1">
				<div
					ref={listRef}
					class="h-full flex flex-col gap-6 overflow-y-auto py-6"
				>
					<header class="space-y-1">
						<h1 class="text-xl font-bold text-zinc-100">Settings</h1>
						<p class="text-sm text-zinc-500">
							Manage your account and how Jota syncs playback.
						</p>
					</header>

					<AccountSettings />
					<RelaySettings />
					<AboutSettings />

					<footer class="mt-auto pt-6 text-center text-xs text-zinc-600">
						Developed by{" "}
						<button
							type="button"
							onClick={openDeveloper}
							class="cursor-pointer text-zinc-400 underline underline-offset-2 transition-colors hover:text-zinc-200"
						>
							@salvadorsru
						</button>
					</footer>
				</div>
				<Scrollbar target={listRef} />
			</div>
		</AppShell>
	);
}

export default SettingsPage;

import type { ComponentChildren } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { DegradedBanner } from "@/lib/auth/views/ui/degraded-banner";
import { SignInPrompt } from "@/lib/auth/views/ui/sign-in-prompt";
import { Welcome } from "@/lib/auth/views/ui/welcome";
import { watch } from "@/lib/clipboard/views/stores/pending";
import { ClipboardBanner } from "@/lib/clipboard/views/ui/banner";
import {
	clearSelection,
	selectionActive,
} from "@/lib/music/views/stores/selection";
import { compactPlayer, PlayerBar } from "@/lib/music/views/ui/player";
import { PlaylistPicker } from "@/lib/music/views/ui/playlists/picker";
import Queue from "@/lib/music/views/ui/queue";
import YoutubeEditor from "@/lib/music/views/ui/track/youtube-editor";
import { cn } from "@/lib/shared/utils/tw";
import {
	Header,
	MobileBottomNav,
} from "@/lib/shared/views/ui/components/header";
import { OverlayHost } from "@/lib/shared/views/ui/components/modal";
import { NotificationBar } from "@/lib/shared/views/ui/notification-bar";
import { SyncPanel } from "@/lib/sync/views/ui/session";
import { UpdateBanner } from "@/lib/update/views/banner";

export function AppShell({
	children,
	player,
	mainClass,
}: {
	children: ComponentChildren;
	player?: ComponentChildren;
	mainClass?: string;
}) {
	const overlayLayer = useRef<HTMLDivElement>(null);
	const [location] = useLocation();

	useEffect(() => {
		clearSelection();
	}, [location]);

	useEffect(() => {
		const onKey = (e: KeyboardEvent) => {
			if (e.key === "Escape" && selectionActive.value) clearSelection();
		};
		document.addEventListener("keydown", onKey);
		return () => document.removeEventListener("keydown", onKey);
	}, []);

	useEffect(() => {
		watch();
	}, []);

	return (
		<OverlayHost.Provider value={overlayLayer}>
			<div class="relative h-screen min-h-0 flex flex-col">
				<div class="flex flex-col bg-stone-950">
					<Header />
					<UpdateBanner />
					<ClipboardBanner />
					<DegradedBanner />
					<SignInPrompt />
				</div>
				<div class="relative flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
					<div
						ref={overlayLayer}
						class="pointer-events-none absolute inset-0 z-50 md:z-120"
					/>

					{player}

					<main
						class={cn(
							"flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl px-4 md:px-0 mx-auto pb-[calc(3.5rem+var(--player-compact)+env(safe-area-inset-bottom))]",
							compactPlayer.value
								? "md:pb-[calc(var(--player-compact)+1.5rem)]"
								: "md:pb-0",
							mainClass,
						)}
					>
						{children}
					</main>

					<PlayerBar />
					<Queue />
					<PlaylistPicker />
					<YoutubeEditor />
					<SyncPanel />
					<Welcome />
				</div>

				<MobileBottomNav />
				<NotificationBar />
			</div>
		</OverlayHost.Provider>
	);
}

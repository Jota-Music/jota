import type { ComponentChildren } from "preact";
import { useEffect, useRef } from "preact/hooks";
import { useLocation } from "wouter-preact";
import { SignInPrompt } from "@/lib/auth/views/ui/sign-in-prompt";
import { Welcome } from "@/lib/auth/views/ui/welcome";
import {
	clearSelection,
	selectionActive,
} from "@/lib/music/views/stores/selection";
import { PlaylistPicker } from "@/lib/music/views/ui/playlists/picker";
import Queue from "@/lib/music/views/ui/queue";
import YoutubeEditor from "@/lib/music/views/ui/track/youtube-editor";
import { cn } from "@/lib/shared/utils/tw";
import { Header } from "@/lib/shared/views/ui/components/header";
import { OverlayHost } from "@/lib/shared/views/ui/components/modal";
import { ErrorBar } from "@/lib/shared/views/ui/error-bar";
import { SyncPanel } from "@/lib/sync/views/ui/session";
import { UpdateBanner } from "@/lib/update/views/banner";

export function AppShell({
	children,
	player,
	rootClass,
	mainClass,
}: {
	children: ComponentChildren;
	player?: ComponentChildren;
	rootClass?: string;
	mainClass?: string;
}) {
	const region = useRef<HTMLDivElement>(null);
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

	return (
		<OverlayHost.Provider value={region}>
			<div class={cn("h-dvh min-h-0 flex flex-col", rootClass)}>
				<div class="flex flex-col bg-stone-950">
					<Header />
					<UpdateBanner />
					<SignInPrompt />
				</div>

				<div
					ref={region}
					class="relative flex min-h-0 flex-1 flex-col gap-4 overflow-hidden"
				>
					{player}

					<main
						class={cn(
							"flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl px-4 md:px-0 mx-auto",
							mainClass,
						)}
					>
						{children}
					</main>

					<Queue />
					<PlaylistPicker />
					<YoutubeEditor />
					<SyncPanel />
					<Welcome />
				</div>

				<ErrorBar />
			</div>
		</OverlayHost.Provider>
	);
}

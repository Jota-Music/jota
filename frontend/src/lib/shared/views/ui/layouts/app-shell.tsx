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
import { backdrop } from "@/lib/music/views/stores/visuals";
import { PlayerBar } from "@/lib/music/views/ui/player";
import { PlaylistPicker } from "@/lib/music/views/ui/playlists/picker";
import Queue from "@/lib/music/views/ui/queue";
import SelectionBar from "@/lib/music/views/ui/track/selection-bar";
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

// Fixed per launch, not per render: a mote that re-rolled its position every
// time the shell painted would jump.
//
// `left` is stratified, one mote per column with a jitter, because independent
// draws clump: at a few dozen points the eye reads the bare patches first. They
// spawn low in the frame because the field rises, and one drawn near the top
// leaves after a few pixels and sits off-screen for most of its cycle, which is
// what made the field look sparse. The peak opacity varies per mote so the
// field has depth instead of reading as one flat sheet. The delay is negative so
// it is already full on the first frame instead of filling in over half a minute.
const DUST = 64;
const dust = Array.from({ length: DUST }, (_, id) => {
	const duration = 18 + Math.random() * 16;
	return {
		id,
		left: `${((id + Math.random()) / DUST) * 100}%`,
		top: `${55 + Math.random() * 45}%`,
		size: `${2 + Math.random() * 2.5}px`,
		peak: `${0.15 + Math.random() * 0.3}`,
		duration,
		delay: -Math.random() * duration,
		sway: Math.random() < 0.5,
	};
});

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
	// The app paints at full opacity; only an arriving page fades in.
	const firstRoute = useRef(location);

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
			<div class="relative grid h-screen min-h-0 grid-rows-[max-content_1fr_max-content]">
				<div class="flex max-w-dvw flex-col bg-stone-950">
					<Header />
					<UpdateBanner />
					<ClipboardBanner />
					<SelectionBar />
					<DegradedBanner />
					<SignInPrompt />
				</div>
				<div
					data-backdrop={backdrop.value || undefined}
					class="relative isolate flex min-h-0 flex-1 flex-col gap-4 overflow-hidden md:p-4 md:px-0 max-w-dvw"
				>
					{backdrop.value && (
						<>
							<div
								aria-hidden="true"
								class="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(circle_at_30%_0%,currentColor,transparent_60%),radial-gradient(circle_at_85%_100%,currentColor,transparent_60%)] text-(--dominant-color) opacity-[0.12] transition-colors duration-1000 ease-out"
							/>

							<div
								aria-hidden="true"
								class="motes pointer-events-none absolute inset-0 -z-10 overflow-hidden text-(--dominant-color) transition-colors duration-1000 ease-out"
							>
								{dust.map((mote) => (
									<span
										key={mote.id}
										class="absolute rounded-full bg-current will-change-[opacity,transform]"
										style={{
											left: mote.left,
											top: mote.top,
											width: mote.size,
											height: mote.size,
											"--peak": mote.peak,
											"--dur": `${mote.duration}s`,
											animation: `${mote.sway ? "mote-sway" : "mote"} ${mote.duration}s linear ${mote.delay}s infinite`,
										}}
									/>
								))}
							</div>
						</>
					)}

					<div
						ref={overlayLayer}
						class="pointer-events-none absolute inset-0 z-50 md:z-120"
					/>

					{player}

					<main
						class={cn(
							"flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl mx-auto",
							mainClass,
						)}
					>
						{/* Keyed on the route so the incoming page fades in instead of
						    cutting: opacity only, so nothing about the geometry moves. */}
						<div
							key={location}
							class={cn(
								"flex min-h-0 flex-1 flex-col",
								location !== firstRoute.current &&
									"animate-[page-in_140ms_ease-out]",
							)}
						>
							{children}
						</div>
					</main>
				</div>

				<div class="flex max-w-dvw flex-col">
					<PlayerBar />
					<MobileBottomNav />
				</div>

				<Queue />
				<PlaylistPicker />
				<YoutubeEditor />
				<SyncPanel />
				<Welcome />
				<NotificationBar />
			</div>
		</OverlayHost.Provider>
	);
}

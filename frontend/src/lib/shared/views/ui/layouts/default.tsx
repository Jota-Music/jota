import type { PropsWithChildren } from "preact/compat";
import { currentSong } from "@/lib/music/views/stores/audio";
import { Player } from "@/lib/music/views/ui/player";
import Queue from "@/lib/music/views/ui/queue";
import { cn } from "@/lib/shared/utils/tw";
import { Header } from "@/lib/shared/views/ui/components/header";
import { ErrorBar } from "@/lib/shared/views/ui/error-bar";
import { SyncPanel } from "@/lib/sync/views/ui/session";

function DefaultLayout({
	children,
	className,
	class: _class,
}: PropsWithChildren & { className?: string; class?: string }) {
	const hasPlayer = !!currentSong.value;
	return (
		<div class={"h-dvh flex flex-col gap-4"}>
			<div
				class={cn(
					"flex flex-col sticky top-0 bg-stone-950",
					hasPlayer && "md:pb-6",
				)}
			>
				<Header />

				<div
					class={cn(
						"shrink-0 w-full max-w-2xl md:mx-auto fixed md:relative",
						hasPlayer && "md:mt-6",
					)}
				>
					<Player />
				</div>
			</div>

			<Queue />

			<main
				class={cn(
					"flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl px-4 md:px-0 mx-auto md:pb-0",
					hasPlayer ? "pb-20" : "pb-4",
					className,
				)}
			>
				{children}
			</main>

			<ErrorBar />
			<SyncPanel />
		</div>
	);
}

export default DefaultLayout;

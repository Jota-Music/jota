import type { PropsWithChildren } from "preact/compat";
import { currentSong } from "@/lib/music/views/stores/audio";
import { compactPlayer, Player } from "@/lib/music/views/ui/player";
import { cn } from "@/lib/shared/utils/tw";
import { AppShell } from "@/lib/shared/views/ui/layouts/app-shell";

function DefaultLayout({
	children,
	className,
}: PropsWithChildren & { className?: string; class?: string }) {
	const hasPlayer = !!currentSong.value;
	const compact = compactPlayer.value;

	return (
		<AppShell
			mainClass={cn(
				hasPlayer ? "pb-20" : "pb-4",
				hasPlayer && compact ? "md:pb-20" : "md:pb-0",
				className,
			)}
			player={
				<div
					class={cn(
						"shrink-0 w-full max-w-2xl md:mx-auto relative",
						hasPlayer && !compact && "md:mt-6 md:pb-6",
					)}
				>
					<Player />
				</div>
			}
		>
			{children}
		</AppShell>
	);
}

export default DefaultLayout;

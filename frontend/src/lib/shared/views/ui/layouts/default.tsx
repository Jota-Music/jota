import type { PropsWithChildren } from "preact/compat";
import { compactPlayer, Player } from "@/lib/music/views/ui/player";
import { cn } from "@/lib/shared/utils/tw";
import { AppShell } from "@/lib/shared/views/ui/layouts/app-shell";

function DefaultLayout({
	children,
	className,
}: PropsWithChildren & { className?: string; class?: string }) {
	const compact = compactPlayer.value;

	return (
		<AppShell
			mainClass={cn("pb-20", compact ? "md:pb-20" : "md:pb-0", className)}
			player={
				<div
					class={cn(
						"shrink-0 w-full max-w-2xl md:mx-auto relative",
						!compact && "md:mt-6 md:pb-6",
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

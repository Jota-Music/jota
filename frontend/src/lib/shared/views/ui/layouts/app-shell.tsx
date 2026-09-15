import type { ComponentChildren } from "preact";
import { useRef } from "preact/hooks";
import { Welcome } from "@/lib/auth/views/ui/welcome";
import Queue from "@/lib/music/views/ui/queue";
import { cn } from "@/lib/shared/utils/tw";
import { Header } from "@/lib/shared/views/ui/components/header";
import { OverlayHost } from "@/lib/shared/views/ui/components/modal";
import { ErrorBar } from "@/lib/shared/views/ui/error-bar";
import { SyncPanel } from "@/lib/sync/views/ui/session";

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

	return (
		<OverlayHost.Provider value={region}>
			<div class={cn("h-dvh min-h-0 flex flex-col", rootClass)}>
				<div class="flex flex-col bg-stone-950">
					<Header />
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
					<SyncPanel />
					<Welcome />
				</div>

				<ErrorBar />
			</div>
		</OverlayHost.Provider>
	);
}

export default AppShell;

import type { PropsWithChildren } from "preact/compat";
import Queue from "@/lib/music/views/ui/queue";
import { cn } from "@/lib/shared/utils/tw";
import { Header } from "@/lib/shared/views/ui/components/header";
import { ErrorBar } from "@/lib/shared/views/ui/error-bar";
import { SyncPanel } from "@/lib/sync/views/ui/session";

function ClearLayout({
	children,
	className,
	class: _class,
}: PropsWithChildren & { className?: string; class?: string }) {
	return (
		<div class={cn("h-dvh min-h-0 flex flex-col pb-6", className, _class)}>
			<Header />

			<Queue />

			<main class="flex min-h-0 flex-1 flex-col min-w-0 overflow-hidden w-full max-w-2xl px-4 md:px-0 mx-auto">
				{children}
			</main>

			<ErrorBar />
			<SyncPanel />
		</div>
	);
}

export default ClearLayout;

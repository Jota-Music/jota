import { Loader } from "lucide-preact";

export function Loading() {
	return (
		<div
			role="status"
			class="flex h-full min-h-0 flex-1 items-center justify-center p-4"
		>
			<Loader size={24} class="animate-spin text-zinc-400" />
		</div>
	);
}

// The song list is drawn inside a frame that the tracks will fill, so the
// placeholder draws the same frame and centres the spinner in it.
export function Rows() {
	return (
		<div class="relative min-h-0 flex-1">
			<div class="h-full overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950 pr-3.75">
				<Loading />
			</div>
		</div>
	);
}

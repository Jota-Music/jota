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
			<div class="h-full overflow-y-auto rounded-none border-t border-x-0 border-b-0 border-zinc-800 md:rounded-lg md:border">
				<Loading />
			</div>
		</div>
	);
}

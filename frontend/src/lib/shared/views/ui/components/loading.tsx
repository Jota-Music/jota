import { useSignal } from "@preact/signals";
import { Loader } from "lucide-preact";
import type { ComponentChildren } from "preact";
import { useEffect } from "preact/hooks";

// A read the store answers in a few milliseconds is not a wait: a placeholder
// for one frame reads as a blink. Whatever stands in for the answer only shows
// once the wait outlives the eye's tolerance, and until then the frame keeps
// its room so nothing jumps when the answer lands.
const flash = 200;

function Delayed({
	delay = flash,
	children,
}: {
	delay?: number;
	children: ComponentChildren;
}) {
	const visible = useSignal(false);

	useEffect(() => {
		const id = setTimeout(() => {
			visible.value = true;
		}, delay);
		return () => clearTimeout(id);
	}, [delay]);

	if (!visible.value) return <div class="h-full min-h-0 flex-1" />;
	return <>{children}</>;
}

export function Loading({ delay }: { delay?: number }) {
	return (
		<Delayed delay={delay}>
			<div
				role="status"
				class="flex h-full min-h-0 flex-1 items-center justify-center p-4"
			>
				<Loader size={24} class="animate-spin text-zinc-400" />
			</div>
		</Delayed>
	);
}

// The song list is drawn inside a frame that the tracks will fill, so the
// placeholder draws the same frame and centres the spinner in it.
export function Rows() {
	return (
		<div class="relative min-h-0 flex-1">
			<div class="h-full overflow-y-auto border-t border-x-0 border-b-0 border-zinc-800 md:rounded-lg md:border">
				<Loading />
			</div>
		</div>
	);
}

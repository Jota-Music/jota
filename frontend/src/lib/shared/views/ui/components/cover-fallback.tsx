import { Music } from "lucide-preact";
import { cn } from "@/lib/shared/utils/tw";

// CoverFallback stands in when there is no artwork. `class` must carry the
// sizing (the same the <img> would have): in a virtualized row the wrapping
// element is sized by the image itself, so a fallback left at `auto` collapses
// to the icon.
export function CoverFallback({ class: className }: { class?: string }) {
	return (
		<div
			class={cn(
				"flex items-center justify-center bg-zinc-900 text-zinc-600",
				className || "h-full w-full",
			)}
		>
			<Music class="h-1/2 max-h-16 w-1/2 min-h-3 min-w-3 max-w-16" />
		</div>
	);
}

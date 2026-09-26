import { Heart } from "lucide-preact";
import { saveLabel, useSaved } from "@/lib/music/views/ui/playlist/saved";
import { cn } from "@/lib/shared/utils/tw";

// cn only joins classes, so the state colours must never overlap with the
// variant's own: the winner would be the CSS source order, not this order. The
// cover sits on artwork and needs a scrim; the header sits on the page
// background, next to the other plain actions.
const base = {
	cover:
		"flex h-8 w-8 cursor-pointer items-center justify-center rounded-md bg-black/70 transition hover:bg-black/90",
	toolbar:
		"flex cursor-pointer items-center justify-center rounded-md p-1.5 transition hover:bg-zinc-800/80",
} as const;

const idle = {
	cover: "text-zinc-300 hover:text-white",
	toolbar: "text-zinc-400 hover:text-white",
} as const;

type Variant = keyof typeof base;

const following = "text-red-400 hover:text-red-300";

// SaveButton follows an external playlist, from either source. It pairs with
// the context menu entry built from the same useSaved hook.
export function SaveButton({
	id,
	variant = "cover",
}: {
	id: string;
	variant?: Variant;
}) {
	const { isSaved, follow } = useSaved();
	const saved = isSaved(id);
	const label = saveLabel(saved);

	return (
		<button
			type="button"
			title={label}
			aria-label={label}
			aria-pressed={saved}
			onClick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				follow(id, !saved);
			}}
			class={cn(base[variant], saved ? following : idle[variant])}
		>
			<Heart size={16} class={saved ? "fill-current" : ""} />
		</button>
	);
}

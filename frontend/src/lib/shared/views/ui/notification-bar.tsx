import { X } from "lucide-preact";
import type { ComponentChildren } from "preact";
import { useRef } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";
import { removeError, roomErrors } from "@/lib/shared/views/stores/errors";

// The single layer of things that arrive on their own (a notice with an action,
// an error): it sits above the player instead of in the page, so one of them
// appearing never pushes the page down. The layer takes no clicks; each notice
// opts back in.
export function NotificationBar({
	children,
}: {
	children?: ComponentChildren;
}) {
	const errors = roomErrors;
	const listRef = useRef<HTMLDivElement>(null);

	return (
		<div class="pointer-events-none fixed bottom-[calc(3.5rem+var(--player-compact)+env(safe-area-inset-bottom))] left-0 right-0 z-60 md:bottom-0">
			<div ref={listRef} class="mx-auto max-w-2xl space-y-2 px-4 py-4">
				{children}
				{errors.value.map((error) => (
					<div
						key={error.id}
						class={cn(
							"animate-[notif-in_0.2s_ease-out] pointer-events-auto flex items-center gap-3 rounded-lg border px-4 py-3 text-sm transition-opacity duration-200",
							error.tone === "info"
								? "border-stone-700 bg-stone-900 text-stone-200"
								: "border-red-900/50 bg-red-950 text-red-200",
							error.leaving && "opacity-0",
						)}
					>
						<div class="flex-1">{error.message}</div>
						<button
							type="button"
							onClick={() => removeError(error.id)}
							class={cn(
								"shrink-0 cursor-pointer transition-colors",
								error.tone === "info"
									? "text-stone-400 hover:text-stone-200"
									: "text-red-400 hover:text-red-200",
							)}
						>
							<X class="size-4" />
						</button>
					</div>
				))}
			</div>
		</div>
	);
}

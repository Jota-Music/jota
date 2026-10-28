import { X } from "lucide-preact";
import { useRef } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";
import { removeError, roomErrors } from "@/lib/shared/views/stores/errors";

export function NotificationBar() {
	const errors = roomErrors;
	const listRef = useRef<HTMLDivElement>(null);

	if (errors.value.length === 0) {
		return null;
	}

	return (
		<div class="fixed bottom-[calc(3.5rem+var(--player-compact)+env(safe-area-inset-bottom))] left-0 right-0 z-60 md:bottom-0">
			<div ref={listRef} class="mx-auto max-w-2xl space-y-2 px-4 py-4">
				{errors.value.map((error) => (
					<div
						key={error.id}
						class={cn(
							"animate-[notif-in_0.2s_ease-out] flex items-center gap-3 rounded-lg border px-4 py-3 text-sm transition-opacity duration-200",
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

import { X } from "lucide-preact";
import { useRef } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";
import { removeError, roomErrors } from "@/lib/shared/views/stores/errors";
import { Scrollbar } from "@/lib/shared/views/ui/components/scrollbar";

export function ErrorBar() {
	const errors = roomErrors;
	const listRef = useRef<HTMLDivElement>(null);

	if (errors.value.length === 0) {
		return null;
	}

	return (
		<div class="fixed bottom-[calc(3.5rem+env(safe-area-inset-bottom))] left-0 right-0 z-60 md:bottom-0">
			<div ref={listRef} class="max-h-96 overflow-y-auto space-y-2 p-4">
				{errors.value.map((error) => (
					<div
						key={error.id}
						class={cn(
							"flex items-center gap-3 rounded-lg border px-4 py-3 text-sm",
							error.tone === "info"
								? "border-stone-700 bg-stone-900 text-stone-200"
								: "border-red-900/50 bg-red-950 text-red-200",
						)}
					>
						<div class="flex-1">{error.message}</div>
						<button
							type="button"
							onClick={() => removeError(error.id)}
							class={cn(
								"shrink-0 transition-colors",
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
			<Scrollbar target={listRef} gutter={16} />
		</div>
	);
}

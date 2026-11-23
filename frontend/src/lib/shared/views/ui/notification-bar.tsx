import { CircleAlert, CircleCheck, X } from "lucide-preact";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";
import { removeError, roomErrors } from "@/lib/shared/views/stores/errors";

export function NotificationBar() {
	const errors = roomErrors;

	if (errors.value.length === 0) {
		return null;
	}

	return (
		// One panel, the sheet's own surface, over the content column and clear of
		// the player bar on every platform: a stack of separate cards read as a
		// second kind of window the rest of the app does not have.
		<div class="pointer-events-none fixed inset-x-0 bottom-[calc(var(--dock-h,0px)+0.75rem)] z-60 flex justify-center px-3">
			<div class="pointer-events-auto max-h-[min(40dvh,18rem)] w-full max-w-2xl divide-y divide-zinc-800/60 overflow-y-auto rounded-2xl border border-zinc-800 bg-zinc-950">
				{errors.value.map((error) => {
					const failed = error.tone === "error";
					return (
						<div
							key={error.id}
							role={failed ? "alert" : "status"}
							class={cn(
								"flex items-start gap-3 px-4 py-3 text-sm",
								error.leaving
									? "animate-[notif-out_160ms_ease-in_forwards]"
									: "animate-[notif-in_200ms_ease-out]",
							)}
						>
							{failed ? (
								<CircleAlert size={16} class="mt-0.5 shrink-0 text-red-400" />
							) : (
								<CircleCheck
									size={16}
									class="mt-0.5 shrink-0 text-(--dominant-color)"
								/>
							)}
							<p class="min-w-0 flex-1 whitespace-pre-line text-pretty leading-snug text-zinc-300">
								{error.message}
							</p>
							<button
								type="button"
								onClick={() => removeError(error.id)}
								aria-label={t("common.dismiss")}
								class="-my-1 shrink-0 cursor-pointer p-1 text-zinc-500 transition-colors hover:text-zinc-200"
							>
								<X size={14} />
							</button>
						</div>
					);
				})}
			</div>
		</div>
	);
}

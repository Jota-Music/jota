import { X } from "lucide-preact";
import { removeError, roomErrors } from "@/lib/shared/views/stores/errors";

export function ErrorBar() {
    const errors = roomErrors;

    if (errors.value.length === 0) {
        return null;
    }

    return (
        <div class="fixed bottom-0 left-0 right-0 z-50 space-y-2 p-4 max-h-96 overflow-y-auto">
            {errors.value.map((error) => (
                <div
                    key={error.id}
                    class="flex items-center gap-3 rounded-lg border border-red-900/50 bg-red-950/80 px-4 py-3 text-sm text-red-200 backdrop-blur"
                >
                    <div class="flex-1">{error.message}</div>
                    <button
                        type="button"
                        onClick={() => removeError(error.id)}
                        class="shrink-0 text-red-400 hover:text-red-200 transition-colors"
                    >
                        <X class="size-4" />
                    </button>
                </div>
            ))}
        </div>
    );
}

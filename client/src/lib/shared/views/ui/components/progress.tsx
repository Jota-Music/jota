import { cn } from "@/lib/shared/utils/tw";
import { useEffect, useRef, useState } from "preact/hooks";

type ProgressProps = {
    value: number;
    min?: number;
    max?: number;
    onChange?: (value: number) => void;
    class?: string;
};

function Progress({
    value,
    min = 0,
    max = 1,
    onChange,
    class: className,
}: ProgressProps) {
    const trackRef = useRef<HTMLDivElement>(null);
    const [dragging, setDragging] = useState(false);

    function update(clientX: number) {
        if (!trackRef.current) return;

        const rect = trackRef.current.getBoundingClientRect();
        const x = clientX - rect.left;

        const ratio = Math.max(0, Math.min(1, x / rect.width));
        const scaledValue = min + ratio * (max - min);

        onChange?.(scaledValue);
    }

    function onMouseDown(e: MouseEvent) {
        setDragging(true);
        update(e.clientX);
    }

    function onMouseMove(e: MouseEvent) {
        if (!dragging) return;
        update(e.clientX);
    }

    function onMouseUp() {
        setDragging(false);
    }

    useEffect(() => {
        window.addEventListener("mousemove", onMouseMove);
        window.addEventListener("mouseup", onMouseUp);

        return () => {
            window.removeEventListener("mousemove", onMouseMove);
            window.removeEventListener("mouseup", onMouseUp);
        };
    }, [dragging]);

    const span = max - min;
    const percent =
        span === 0 || !Number.isFinite(span)
            ? 0
            : Math.min(100, Math.max(0, ((value - min) / span) * 100));

    return (
        <div
            ref={trackRef}
            onMouseDown={onMouseDown}
            class={cn(
                "relative h-2 w-full cursor-pointer select-none rounded-full bg-neutral-300",
                className
            )}
        >
            {/* Fill */}
            <div
                class="absolute left-0 top-0 h-full bg-current rounded-full"
                style={{
                    width: `${percent}%`,
                }}
            />

            {/* Thumb */}
            <div
                class={cn(
                    "absolute top-1/2 -translate-y-1/2 -translate-x-1/2 w-3 h-3 rounded-full bg-current transition-transform",
                    dragging ? "scale-110 cursor-grabbing" : "cursor-grab"
                )}
                style={{
                    left: `${percent}%`,
                }}
            />
        </div>
    );
}

export default Progress;
import { useEffect, useRef, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

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

	function getClientX(e: MouseEvent | TouchEvent): number {
		return "touches" in e ? e.touches[0].clientX : e.clientX;
	}

	function update(clientX: number) {
		if (!trackRef.current) return;

		const rect = trackRef.current.getBoundingClientRect();
		const x = clientX - rect.left;

		const ratio = Math.max(0, Math.min(1, x / rect.width));
		const scaledValue = min + ratio * (max - min);

		onChange?.(scaledValue);
	}

	function onPointerDown(e: MouseEvent | TouchEvent) {
		setDragging(true);
		update(getClientX(e));
	}

	function onPointerMove(e: MouseEvent | TouchEvent) {
		if (!dragging) return;
		e.preventDefault();
		update(getClientX(e));
	}

	function onPointerUp() {
		setDragging(false);
	}

	function onWheel(e: WheelEvent) {
		e.preventDefault();
		const step = (max - min) * 0.05;
		const next = value + (e.deltaY < 0 ? step : -step);
		onChange?.(Math.max(min, Math.min(max, next)));
	}

	useEffect(() => {
		window.addEventListener("mousemove", onPointerMove);
		window.addEventListener("mouseup", onPointerUp);
		window.addEventListener("touchmove", onPointerMove, { passive: false });
		window.addEventListener("touchend", onPointerUp);

		return () => {
			window.removeEventListener("mousemove", onPointerMove);
			window.removeEventListener("mouseup", onPointerUp);
			window.removeEventListener("touchmove", onPointerMove);
			window.removeEventListener("touchend", onPointerUp);
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
			onMouseDown={onPointerDown}
			onTouchStart={onPointerDown}
			onWheel={onWheel}
			class={cn(
				"relative h-2 w-full cursor-pointer select-none rounded-full bg-neutral-300 touch-none",
				className,
			)}
			role="slider"
			tabIndex={0}
			aria-valuenow={value}
			aria-valuemin={min}
			aria-valuemax={max}
			aria-label="Seek"
		>
			<div
				class="absolute left-0 top-0 h-full bg-current rounded-full"
				style={{
					width: `${percent}%`,
				}}
			/>

			<div
				class={cn(
					"absolute top-1/2 -translate-y-1/2 -translate-x-1/2 w-3 h-3 rounded-full bg-current transition-transform",
					dragging ? "scale-110 cursor-grabbing" : "cursor-grab",
				)}
				style={{
					left: `${percent}%`,
				}}
			/>
		</div>
	);
}

export default Progress;

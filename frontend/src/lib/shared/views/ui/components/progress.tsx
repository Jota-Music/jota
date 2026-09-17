import { useEffect, useRef, useState } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

type ProgressProps = {
	value: number;
	min?: number;
	max?: number;
	onChange?: (value: number) => void;
	onCommit?: (value: number) => void;
	class?: string;
};

function clientXOf(e: MouseEvent | TouchEvent): number {
	if ("touches" in e) {
		const touch = e.touches[0] ?? e.changedTouches[0];
		return touch ? touch.clientX : 0;
	}
	return e.clientX;
}

function Progress({
	value,
	min = 0,
	max = 1,
	onChange,
	onCommit,
	class: className,
}: ProgressProps) {
	const trackRef = useRef<HTMLDivElement>(null);
	const draggingRef = useRef(false);
	const lastValue = useRef(0);
	const [dragging, setDragging] = useState(false);
	const propsRef = useRef({ onChange, onCommit, min, max });
	propsRef.current = { onChange, onCommit, min, max };

	function preview(clientX: number) {
		const track = trackRef.current;
		if (!track) return;
		const { min, max, onChange } = propsRef.current;
		const rect = track.getBoundingClientRect();
		const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
		const scaled = min + ratio * (max - min);
		lastValue.current = scaled;
		onChange?.(scaled);
	}

	// Global listeners are attached once. They read the drag state from a ref,
	// so a fast tap (or an Android touchcancel) can never miss the release and
	// leave the bar stuck.
	useEffect(() => {
		function move(e: MouseEvent | TouchEvent) {
			if (!draggingRef.current) return;
			if ("touches" in e) e.preventDefault();
			preview(clientXOf(e));
		}
		function finish() {
			if (!draggingRef.current) return;
			draggingRef.current = false;
			setDragging(false);
			propsRef.current.onCommit?.(lastValue.current);
		}

		window.addEventListener("mousemove", move);
		window.addEventListener("mouseup", finish);
		window.addEventListener("touchmove", move, { passive: false });
		window.addEventListener("touchend", finish);
		window.addEventListener("touchcancel", finish);

		return () => {
			window.removeEventListener("mousemove", move);
			window.removeEventListener("mouseup", finish);
			window.removeEventListener("touchmove", move);
			window.removeEventListener("touchend", finish);
			window.removeEventListener("touchcancel", finish);
		};
	}, []);

	function start(e: MouseEvent | TouchEvent) {
		draggingRef.current = true;
		setDragging(true);
		preview(clientXOf(e));
	}

	function onWheel(e: WheelEvent) {
		e.preventDefault();
		const step = (max - min) * 0.05;
		const next = value + (e.deltaY < 0 ? step : -step);
		const clamped = Math.max(min, Math.min(max, next));
		lastValue.current = clamped;
		if (onCommit) {
			onCommit(clamped);
		} else {
			onChange?.(clamped);
		}
	}

	const span = max - min;
	const percent =
		span === 0 || !Number.isFinite(span)
			? 0
			: Math.min(100, Math.max(0, ((value - min) / span) * 100));

	return (
		<div
			ref={trackRef}
			onMouseDown={start}
			onTouchStart={start}
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
			aria-label={t("music.player.seek")}
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

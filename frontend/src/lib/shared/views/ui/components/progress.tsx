import { useSignal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { clientAxis } from "@/lib/shared/utils/pointer";
import { cn } from "@/lib/shared/utils/tw";

type ProgressProps = {
	value: number;
	min?: number;
	max?: number;
	onChange?: (value: number) => void;
	onCommit?: (value: number) => void;
	class?: string;
	disabled?: boolean;
	// A vertical bar fills from the bottom, so it reads as a fader.
	vertical?: boolean;
	label?: string;
};

function Progress({
	value,
	min = 0,
	max = 1,
	onChange,
	onCommit,
	class: className,
	disabled = false,
	vertical = false,
	label,
}: ProgressProps) {
	const trackRef = useRef<HTMLDivElement>(null);
	const dragging = useSignal(false);
	// Null until this drag has produced a position: a release with nothing to
	// commit must not seek to a value the user never picked.
	const lastValue = useRef<number | null>(null);
	const propsRef = useRef({ onChange, onCommit, min, max, vertical });
	propsRef.current = { onChange, onCommit, min, max, vertical };

	function preview(client: number) {
		const track = trackRef.current;
		if (!track) return;
		const { min, max, onChange, vertical } = propsRef.current;
		const rect = track.getBoundingClientRect();
		// A bar with no size cannot be read as a position: dividing by it gives
		// Infinity, which clamps to the far end and seeks the track to its last
		// frame. Nothing the user picked, so pick nothing.
		const size = vertical ? rect.height : rect.width;
		if (!(size > 0)) return;
		const ratio = vertical
			? 1 - (client - rect.top) / size
			: (client - rect.left) / size;
		const clamped = Math.max(0, Math.min(1, ratio));
		const scaled = min + clamped * (max - min);
		lastValue.current = scaled;
		onChange?.(scaled);
	}

	// Global listeners are attached once. They read the drag state from a ref,
	// so a fast tap (or an Android touchcancel) can never miss the release and
	// leave the bar stuck.
	useEffect(() => {
		function move(e: MouseEvent | TouchEvent) {
			if (!dragging.value) return;
			if ("touches" in e) e.preventDefault();
			preview(clientAxis(e, propsRef.current.vertical ? "y" : "x"));
		}
		function finish() {
			if (!dragging.value) return;
			dragging.value = false;
			if (lastValue.current === null) return;
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
		if (disabled) return;
		dragging.value = true;
		lastValue.current = null;
		preview(clientAxis(e, vertical ? "y" : "x"));
	}

	function onWheel(e: WheelEvent) {
		if (disabled) return;
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
				"relative select-none rounded-full bg-neutral-800 touch-none",
				vertical ? "h-24 w-2" : "h-2 w-full",
				disabled
					? "cursor-default brightness-75 saturate-50"
					: "cursor-pointer",
				className,
			)}
			role="slider"
			tabIndex={disabled ? -1 : 0}
			aria-disabled={disabled}
			aria-orientation={vertical ? "vertical" : "horizontal"}
			aria-valuenow={value}
			aria-valuemin={min}
			aria-valuemax={max}
			aria-label={label ?? t("music.player.seek")}
		>
			<div
				class="absolute rounded-full bg-current"
				style={
					vertical
						? { left: 0, bottom: 0, width: "100%", height: `${percent}%` }
						: { left: 0, top: 0, height: "100%", width: `${percent}%` }
				}
			/>

			<div
				class={cn(
					"absolute h-3 w-3 rounded-full bg-current transition-transform",
					vertical
						? "bottom-0 left-1/2 -translate-x-1/2 translate-y-1/2"
						: "top-1/2 -translate-x-1/2 -translate-y-1/2",
					dragging.value ? "scale-110 cursor-grabbing" : "cursor-grab",
				)}
				style={vertical ? { bottom: `${percent}%` } : { left: `${percent}%` }}
			/>
		</div>
	);
}

export default Progress;

import { useSignal, useSignalEffect } from "@preact/signals";
import type { PropsWithChildren } from "preact/compat";
import { useEffect, useRef } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

const STROKE_WIDTH = 8;

type CircularProgressProps = {
	value: number;
	min?: number;
	max?: number;
	onChange?: (value: number) => void;
	onCommit?: (value: number) => void;
	class?: string;
} & PropsWithChildren;

function CircularProgress({
	value,
	min = 0,
	max = 1,
	onChange,
	onCommit,
	class: className,
	children,
}: CircularProgressProps) {
	const containerRef = useRef<HTMLDivElement>(null);
	const svgRef = useRef<SVGSVGElement>(null);
	const lastValue = useRef(0);

	const size = useSignal(0);
	const dragging = useSignal(false);
	const onRing = useSignal(false);

	const propsRef = useRef({ min, max, onChange, onCommit });
	propsRef.current = { min, max, onChange, onCommit };

	useEffect(() => {
		if (!containerRef.current) return;

		const el = containerRef.current;

		const update = () => {
			const rect = el.getBoundingClientRect();
			size.value = rect.width || 0;
		};

		update();

		const observer = new ResizeObserver(update);
		observer.observe(el);

		return () => observer.disconnect();
	}, []);

	const safeSize = Math.max(0, size.value);
	const center = safeSize / 2;

	const radius = Math.max(0, (safeSize - STROKE_WIDTH) / 2);
	const circumference = 2 * Math.PI * radius;

	const percent = max !== min ? (value - min) / (max - min) : 0;
	const clamped = Math.min(1, Math.max(0, percent));
	const offset = circumference * (1 - clamped);

	function updateFromPointer(clientX: number, clientY: number) {
		const svg = svgRef.current;
		if (!svg || size.value === 0) return;

		const rect = svg.getBoundingClientRect();
		const cx = rect.left + rect.width / 2;
		const cy = rect.top + rect.height / 2;

		let angle = Math.atan2(clientY - cy, clientX - cx);
		if (angle < 0) angle += Math.PI * 2;

		// align -90deg
		angle = (angle + Math.PI / 2) % (Math.PI * 2);

		const { min, max, onChange } = propsRef.current;
		const ratio = angle / (Math.PI * 2);
		const next = Math.min(max, Math.max(min, min + ratio * (max - min)));

		lastValue.current = next;
		onChange?.(next);
	}

	function isOnRing(clientX: number, clientY: number) {
		const svg = svgRef.current;
		if (!svg || size.value === 0) return false;

		const rect = svg.getBoundingClientRect();
		const cx = rect.left + rect.width / 2;
		const cy = rect.top + rect.height / 2;

		const dx = clientX - cx;
		const dy = clientY - cy;
		const dist = Math.sqrt(dx * dx + dy * dy);

		const inner = radius - STROKE_WIDTH / 2;
		const outer = radius + STROKE_WIDTH / 2;

		return dist >= inner && dist <= outer;
	}

	// Drag listeners are attached once and read state from refs, so a fast tap
	// or an Android pointercancel can never miss the release and freeze the bar.
	useEffect(() => {
		const move = (e: PointerEvent) => {
			if (!dragging.value) return;
			updateFromPointer(e.clientX, e.clientY);
		};

		const finish = () => {
			if (!dragging.value) return;
			dragging.value = false;
			propsRef.current.onCommit?.(lastValue.current);
		};

		window.addEventListener("pointermove", move);
		window.addEventListener("pointerup", finish);
		window.addEventListener("pointercancel", finish);

		return () => {
			window.removeEventListener("pointermove", move);
			window.removeEventListener("pointerup", finish);
			window.removeEventListener("pointercancel", finish);
		};
	}, []);

	function onPointerDown(e: PointerEvent) {
		if (!isOnRing(e.clientX, e.clientY)) return;

		dragging.value = true;
		onRing.value = true;

		updateFromPointer(e.clientX, e.clientY);

		(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId);
	}

	function onHoverMove(e: PointerEvent) {
		if (dragging.value) return;
		onRing.value = isOnRing(e.clientX, e.clientY);
	}

	useSignalEffect(() => {
		document.body.style.cursor = dragging.value
			? "grabbing"
			: onRing.value
				? "grab"
				: "";
		return () => {
			document.body.style.cursor = "";
		};
	});

	return (
		<div
			ref={containerRef}
			class={cn(
				"relative inline-flex aspect-square shrink-0 select-none touch-none",
				className,
			)}
			role="slider"
			tabIndex={0}
			aria-valuenow={value}
			aria-valuemin={min}
			aria-valuemax={max}
			onPointerDown={onPointerDown}
			onPointerMove={onHoverMove}
			onPointerLeave={() => {
				onRing.value = false;
			}}
		>
			{/* CENTER */}
			<div class="absolute inset-0 flex items-center justify-center overflow-hidden rounded-full z-0 pointer-events-auto">
				{children}
			</div>

			{/* SVG */}
			<svg
				ref={svgRef}
				width="100%"
				height="100%"
				viewBox={`0 0 ${safeSize} ${safeSize}`}
				class="relative z-10 pointer-events-none"
			>
				<title>Circular Progress</title>

				{/* Background */}
				<circle
					cx={center}
					cy={center}
					r={radius}
					stroke="currentColor"
					stroke-opacity="0.2"
					stroke-width={STROKE_WIDTH}
					fill="none"
					class="drop-shadow-lg drop-shadow-black"
				/>

				{/* Progress */}
				<circle
					cx={center}
					cy={center}
					r={radius}
					stroke="currentColor"
					stroke-width={STROKE_WIDTH}
					fill="none"
					stroke-linecap="round"
					stroke-dasharray={circumference}
					stroke-dashoffset={offset}
					transform={`rotate(-90 ${center} ${center})`}
					class="text-(--primary,currentColor) "
				/>
			</svg>
		</div>
	);
}

export default CircularProgress;

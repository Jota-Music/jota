import type { CSSProperties } from "preact";
import type { PropsWithChildren } from "preact/compat";
import { useEffect, useRef, useState } from "preact/hooks";
import { cn } from "@/lib/shared/utils/tw";

type CircularProgressProps = {
	value: number;
	min?: number;
	max?: number;
	strokeWidth?: number;
	onChange?: (value: number) => void;
	onCommit?: (value: number) => void;
	class?: string;
	style?: CSSProperties;
} & PropsWithChildren;

function CircularProgress({
	value,
	min = 0,
	max = 1,
	strokeWidth = 8,
	onChange,
	onCommit,
	class: className,
	children,
	style,
}: CircularProgressProps) {
	const containerRef = useRef<HTMLDivElement>(null);
	const svgRef = useRef<SVGSVGElement>(null);
	const draggingRef = useRef(false);
	const lastValue = useRef(0);

	const [size, setSize] = useState(0);
	const [dragging, setDragging] = useState(false);
	const [onRing, setOnRing] = useState(false);

	const sizeRef = useRef(0);
	sizeRef.current = size;
	const propsRef = useRef({ min, max, onChange, onCommit });
	propsRef.current = { min, max, onChange, onCommit };

	useEffect(() => {
		if (!containerRef.current) return;

		const el = containerRef.current;

		const update = () => {
			const rect = el.getBoundingClientRect();
			setSize(rect.width || 0);
		};

		update();

		const observer = new ResizeObserver(update);
		observer.observe(el);

		return () => observer.disconnect();
	}, []);

	const safeSize = Math.max(0, size);
	const center = safeSize / 2;

	const radius = Math.max(0, (safeSize - strokeWidth) / 2);
	const circumference = 2 * Math.PI * radius;

	const percent = max !== min ? (value - min) / (max - min) : 0;
	const clamped = Math.min(1, Math.max(0, percent));
	const offset = circumference * (1 - clamped);

	function updateFromPointer(clientX: number, clientY: number) {
		const svg = svgRef.current;
		if (!svg || sizeRef.current === 0) return;

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
		if (!svg || sizeRef.current === 0) return false;

		const rect = svg.getBoundingClientRect();
		const cx = rect.left + rect.width / 2;
		const cy = rect.top + rect.height / 2;

		const dx = clientX - cx;
		const dy = clientY - cy;
		const dist = Math.sqrt(dx * dx + dy * dy);

		const inner = radius - strokeWidth / 2;
		const outer = radius + strokeWidth / 2;

		return dist >= inner && dist <= outer;
	}

	// Drag listeners are attached once and read state from refs, so a fast tap
	// or an Android pointercancel can never miss the release and freeze the bar.
	useEffect(() => {
		const move = (e: PointerEvent) => {
			if (!draggingRef.current) return;
			updateFromPointer(e.clientX, e.clientY);
		};

		const finish = () => {
			if (!draggingRef.current) return;
			draggingRef.current = false;
			setDragging(false);
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

		draggingRef.current = true;
		setDragging(true);
		setOnRing(true);

		updateFromPointer(e.clientX, e.clientY);

		(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId);
	}

	function onHoverMove(e: PointerEvent) {
		if (draggingRef.current) return;
		setOnRing(isOnRing(e.clientX, e.clientY));
	}

	useEffect(() => {
		if (dragging) {
			document.body.style.cursor = "grabbing";
		} else if (onRing) {
			document.body.style.cursor = "grab";
		} else {
			document.body.style.cursor = "";
		}

		return () => {
			document.body.style.cursor = "";
		};
	}, [dragging, onRing]);

	return (
		<div
			ref={containerRef}
			class={cn(
				"relative inline-flex aspect-square shrink-0 select-none touch-none",
				className,
			)}
			style={style}
			role="slider"
			tabIndex={0}
			aria-valuenow={value}
			aria-valuemin={min}
			aria-valuemax={max}
			onPointerDown={onPointerDown}
			onPointerMove={onHoverMove}
			onPointerLeave={() => setOnRing(false)}
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
					stroke-width={strokeWidth}
					fill="none"
					class="drop-shadow-lg drop-shadow-black"
				/>

				{/* Progress */}
				<circle
					cx={center}
					cy={center}
					r={radius}
					stroke="currentColor"
					stroke-width={strokeWidth}
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

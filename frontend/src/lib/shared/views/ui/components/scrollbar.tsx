import type { RefObject } from "preact";
import { useLayoutEffect, useRef } from "preact/hooks";

const MIN_THUMB_PX = 40;
const CORNER_PX = 2;

type Metrics = { height: number; travel: number; max: number; inset: number };

// gutter is the right padding the scroller already reserves in CSS; the thumb
// only needs to align to it, never to write layout itself.
export function Scrollbar({
	target,
	gutter = 15,
}: {
	target: RefObject<HTMLElement>;
	gutter?: number;
}) {
	const bar = useRef<HTMLDivElement>(null);
	const drag = useRef<{ y: number; top: number } | null>(null);
	const metrics = useRef<Metrics>({ height: 0, travel: 0, max: 0, inset: 0 });

	useLayoutEffect(() => {
		const node = bar.current;
		if (!node) return;

		const m = metrics.current;
		let frame = 0;
		let watch = 0;
		let el: HTMLElement;
		let detach: (() => void) | undefined;

		const place = () => {
			frame = 0;
			const ratio = m.max > 0 ? el.scrollTop / m.max : 0;
			node.style.transform = `translateY(${m.inset + ratio * m.travel}px)`;
		};

		const sync = () => {
			if (frame === 0) frame = requestAnimationFrame(place);
		};

		const measure = () => {
			const inset = el.clientTop > 0 ? CORNER_PX : 0;
			const track = el.offsetHeight - inset * 2;
			const { scrollHeight, clientHeight } = el;
			if (clientHeight === 0 || scrollHeight <= clientHeight || track <= 0) {
				m.height = 0;
				m.travel = 0;
				m.max = 0;
				m.inset = 0;
				node.style.height = "0px";
			} else {
				const ratio = clientHeight / scrollHeight;
				m.height = Math.min(track, Math.max(MIN_THUMB_PX, ratio * track));
				m.travel = track - m.height;
				m.max = scrollHeight - clientHeight;
				m.inset = inset;
				node.style.height = `${m.height}px`;
			}
			place();
		};

		const sizes = new ResizeObserver(measure);
		const observeContent = () => {
			sizes.disconnect();
			sizes.observe(el);
			for (const child of el.children) sizes.observe(child);
			measure();
		};

		const attach = () => {
			const scroller = target.current;
			// The scroller can mount a frame later, as in a modal animating in.
			// This effect only reruns when `target` changes identity, so bailing
			// here would leave the observers off for good.
			if (!scroller) {
				watch = requestAnimationFrame(attach);
				return;
			}
			el = scroller;

			const mutations = new MutationObserver(observeContent);
			mutations.observe(el, { childList: true });
			observeContent();

			el.addEventListener("scroll", sync, { passive: true });
			detach = () => {
				mutations.disconnect();
				sizes.disconnect();
				el.removeEventListener("scroll", sync);
			};
		};

		attach();

		return () => {
			if (frame !== 0) cancelAnimationFrame(frame);
			if (watch !== 0) cancelAnimationFrame(watch);
			detach?.();
		};
	}, [target]);

	const onDown = (e: PointerEvent) => {
		const el = target.current;
		if (!el) return;
		drag.current = { y: e.clientY, top: el.scrollTop };
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		e.preventDefault();
	};

	const onMove = (e: PointerEvent) => {
		const el = target.current;
		const start = drag.current;
		const m = metrics.current;
		if (!el || !start || m.travel <= 0) return;
		const next = start.top + ((e.clientY - start.y) / m.travel) * m.max;
		el.scrollTop = Math.max(0, Math.min(m.max, next));
	};

	const onUp = (e: PointerEvent) => {
		if (!drag.current) return;
		drag.current = null;
		const node = e.currentTarget as HTMLElement;
		if (node.hasPointerCapture(e.pointerId)) {
			node.releasePointerCapture(e.pointerId);
		}
	};

	return (
		<div
			ref={bar}
			aria-hidden
			class="absolute right-0 top-0 w-6 touch-none cursor-grab active:cursor-grabbing"
			onPointerDown={onDown}
			onPointerMove={onMove}
			onPointerUp={onUp}
			onPointerCancel={onUp}
			onLostPointerCapture={onUp}
		>
			<div
				class="ml-auto h-full w-2 rounded-full bg-(--scrollbar-thumb) transition-colors pointer-fine:hover:bg-(--scrollbar-thumb-hover)"
				style={{ marginRight: `${(gutter - 8) / 2}px` }}
			/>
		</div>
	);
}

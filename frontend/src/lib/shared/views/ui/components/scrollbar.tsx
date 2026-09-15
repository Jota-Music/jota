import type { RefObject } from "preact";
import { useEffect, useRef } from "preact/hooks";

const MIN_THUMB_PX = 40;

function thumbFor(el: HTMLElement) {
	const { scrollTop, scrollHeight, clientHeight } = el;
	if (clientHeight === 0 || scrollHeight <= clientHeight) return null;
	const ratio = clientHeight / scrollHeight;
	const height = Math.min(
		clientHeight,
		Math.max(MIN_THUMB_PX, ratio * clientHeight),
	);
	const top =
		(scrollTop / (scrollHeight - clientHeight)) * (clientHeight - height);
	return { top, height };
}

export function Scrollbar({ target }: { target: RefObject<HTMLElement> }) {
	const bar = useRef<HTMLDivElement>(null);
	const drag = useRef<{ y: number; top: number } | null>(null);

	useEffect(() => {
		if (window.matchMedia("(pointer: fine)").matches) return;
		const el = target.current;
		const node = bar.current;
		if (!el || !node) return;

		const sync = () => {
			const thumb = thumbFor(el);
			node.style.top = `${thumb?.top ?? 0}px`;
			node.style.height = `${thumb?.height ?? 0}px`;
		};

		const sizes = new ResizeObserver(sync);
		const observeContent = () => {
			sizes.disconnect();
			sizes.observe(el);
			for (const child of el.children) sizes.observe(child);
			sync();
		};

		const mutations = new MutationObserver(observeContent);
		mutations.observe(el, { childList: true });
		observeContent();

		el.addEventListener("scroll", sync, { passive: true });
		return () => {
			el.removeEventListener("scroll", sync);
			sizes.disconnect();
			mutations.disconnect();
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
		if (!el || !start) return;
		const thumb = thumbFor(el);
		if (!thumb) return;
		const travel = el.clientHeight - thumb.height;
		if (travel <= 0) return;
		const max = el.scrollHeight - el.clientHeight;
		const next = start.top + ((e.clientY - start.y) / travel) * max;
		el.scrollTop = Math.max(0, Math.min(max, next));
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
			class="absolute right-0 w-6 touch-none pointer-fine:hidden"
			onPointerDown={onDown}
			onPointerMove={onMove}
			onPointerUp={onUp}
			onPointerCancel={onUp}
			onLostPointerCapture={onUp}
		>
			<div class="ml-auto mr-1 h-full w-1.5 rounded-full bg-(--scrollbar-thumb)" />
		</div>
	);
}

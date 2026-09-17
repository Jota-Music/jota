import { useLayoutEffect, useRef, useState } from "preact/hooks";

interface WindowItem {
	index: number;
	start: number;
	size: number;
}

// Fixed-height list windowing: reports the rows intersecting the viewport plus
// overscan. Only the rows it returns should be rendered, absolutely positioned
// at `start`. The element tracked by the returned ref may mount late (e.g. a
// closed modal), so it is watched per render instead of once.
export function useWindow<T extends HTMLElement>(
	count: number,
	rowHeight: number,
	overscan = 5,
) {
	const ref = useRef<T>(null);
	const [element, setElement] = useState<T | null>(null);
	const [scrollTop, setScrollTop] = useState(0);
	const [viewport, setViewport] = useState(0);

	useLayoutEffect(() => {
		if (ref.current !== element) setElement(ref.current);
	});

	useLayoutEffect(() => {
		const el = element;
		if (!el) return;

		let frame = 0;
		const sync = () => {
			if (frame !== 0) return;
			frame = requestAnimationFrame(() => {
				frame = 0;
				setScrollTop(el.scrollTop);
				setViewport(el.clientHeight);
			});
		};

		const sizes = new ResizeObserver(sync);
		sizes.observe(el);
		el.addEventListener("scroll", sync, { passive: true });
		sync();
		return () => {
			if (frame !== 0) cancelAnimationFrame(frame);
			sizes.disconnect();
			el.removeEventListener("scroll", sync);
		};
	}, [element]);

	const first = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan);
	const visible = Math.ceil(viewport / rowHeight) + overscan * 2;
	const last = Math.min(count, first + visible);

	const items: WindowItem[] = [];
	for (let index = first; index < last; index++) {
		items.push({ index, start: index * rowHeight, size: rowHeight });
	}

	return { ref, totalSize: count * rowHeight, items };
}

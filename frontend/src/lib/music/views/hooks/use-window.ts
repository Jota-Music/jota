import { useLayoutEffect, useRef, useState } from "preact/hooks";

interface WindowItem {
	index: number;
	start: number;
	size: number;
}

interface Band {
	first: number;
	visible: number;
}

// The band is what the list renders, so the same scroll offset yields the same
// band: a scroll that stays inside it touches nothing.
export function band(
	top: number,
	height: number,
	rowHeight: number,
	overscan: number,
): Band {
	return {
		first: Math.max(0, Math.floor(top / rowHeight) - overscan),
		visible: Math.ceil(height / rowHeight) + overscan * 2,
	};
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
	// The window, not the offset: the rows sit absolutely inside a spacer of the
	// full height, so scrolling inside the same band moves them natively and
	// needs no render. Keeping the offset here would rerender the list for every
	// frame of a scroll to produce the very same rows.
	const [view, setView] = useState<Band>({ first: 0, visible: overscan * 2 });

	useLayoutEffect(() => {
		if (ref.current !== element) setElement(ref.current);
	});

	useLayoutEffect(() => {
		const el = element;
		if (!el) return;

		let frame = 0;
		const read = () => {
			const height = el.clientHeight;
			// A hidden or not-yet-measured pane reports no height: windowing to the
			// overscan alone would paint the top rows and leave the rest blank.
			if (height === 0) return;
			const next = band(el.scrollTop, height, rowHeight, overscan);
			setView((previous) =>
				previous.first === next.first && previous.visible === next.visible
					? previous
					: next,
			);
		};
		// Scrolling and resizing are throttled to a frame, but the first read is
		// not: a viewport of 0 would window the list to the overscan alone and
		// then grow it a frame later, right after the first paint.
		const sync = () => {
			if (frame !== 0) return;
			frame = requestAnimationFrame(() => {
				frame = 0;
				read();
			});
		};

		const sizes = new ResizeObserver(sync);
		sizes.observe(el);
		el.addEventListener("scroll", sync, { passive: true });
		read();
		return () => {
			if (frame !== 0) cancelAnimationFrame(frame);
			sizes.disconnect();
			el.removeEventListener("scroll", sync);
		};
	}, [element, rowHeight, overscan]);

	const last = Math.min(count, view.first + view.visible);

	const items: WindowItem[] = [];
	for (let index = view.first; index < last; index++) {
		items.push({ index, start: index * rowHeight, size: rowHeight });
	}

	return { ref, totalSize: count * rowHeight, items };
}

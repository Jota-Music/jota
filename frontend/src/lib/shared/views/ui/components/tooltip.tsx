import { signal } from "@preact/signals";
import { createPortal } from "preact/compat";
import { useEffect, useLayoutEffect, useRef } from "preact/hooks";

// One tooltip for the whole app. It reads what the control already carries: a
// title, or the accessible name of a button or link that has no title. The title
// is moved to `data-tip` the moment the pointer or the keyboard lands on it, so
// the platform's own tooltip (slow, unstyled) never fires. Nothing is wrapped and
// nothing is measured at render time, so a heart over artwork or a button inside
// a scrolling row gets the same tip as one in a header, with no clipping.
const GAP = 6;
const EDGE = 8;

// A control answers with its own name; a region or a card does not.
const CONTROL = "a,button,[role='button'],[role='menuitem']";

type Shown = { label: string; x: number; y: number; anchorTop: number };

const tip = signal<Shown | null>(null);

const coarse = () => window.matchMedia("(pointer: coarse)").matches;

function adopt(mark: Element) {
	const title = mark.getAttribute("title");
	if (title === null) return;
	const previous = mark.getAttribute("data-tip");
	mark.setAttribute("data-tip", title);
	// A title is often the only name a control has, so it stays available to a
	// screen reader once it is off the platform's tooltip path. A label the
	// author wrote is left alone, and one this component set is replaced when
	// the title changes: a pin that becomes unpin must not keep the old name.
	const label = mark.getAttribute("aria-label");
	if (label === null || label === previous)
		mark.setAttribute("aria-label", title);
	mark.removeAttribute("title");
}

function labelOf(mark: Element) {
	return (
		mark.getAttribute("data-tip") ??
		mark.getAttribute("title") ??
		(mark.matches(CONTROL) ? mark.getAttribute("aria-label") : null)
	);
}

function owner(node: EventTarget | null) {
	if (!(node instanceof Element)) return null;
	// Walk up rather than stopping at the nearest mark: an icon can carry an
	// aria-label of its own ("YouTube icon"), and the control around it is the
	// thing with a name worth showing. A region or a card with an aria-label
	// explains nothing on hover, so only a node with a usable label answers.
	for (let el: Element | null = node; el; el = el.parentElement) {
		if (labelOf(el)) return el;
	}
	return null;
}

function show(mark: Element) {
	adopt(mark);
	const label = labelOf(mark);
	if (!label) return;
	const box = mark.getBoundingClientRect();
	tip.value = {
		label,
		x: box.left + box.width / 2,
		y: box.bottom + GAP,
		anchorTop: box.top,
	};
}

const hide = () => {
	clearTimeout(timer);
	tip.value = null;
};

// Long enough that sweeping the mouse across a row of controls shows nothing,
// short enough that a deliberate hover feels answered.
const DELAY = 250;
let timer: ReturnType<typeof setTimeout> | undefined;

function showSoon(mark: Element) {
	clearTimeout(timer);
	timer = setTimeout(() => {
		show(mark);
	}, DELAY);
}

export function Tooltip() {
	const tipRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		const over = (event: PointerEvent) => {
			const mark = owner(event.target);
			if (mark) {
				showSoon(mark);
			} else {
				hide();
			}
		};
		const enter = (event: FocusEvent) => {
			// Touch has no hover, and a tip that stays until the next tap is noise.
			if (coarse()) return;
			const focused = event.target instanceof Element ? event.target : null;
			// A click also focuses, and a tip left behind by the click that just
			// landed is in the way: only the keyboard asks for one.
			if (!focused?.matches(":focus-visible")) return;
			const mark = owner(focused);
			if (mark) show(mark);
		};
		const key = (event: KeyboardEvent) => {
			if (event.key === "Escape") hide();
		};

		document.addEventListener("pointerover", over);
		document.addEventListener("pointerdown", hide);
		document.addEventListener("focusin", enter);
		document.addEventListener("focusout", hide);
		document.addEventListener("keydown", key);
		document.documentElement.addEventListener("pointerleave", hide);
		window.addEventListener("scroll", hide, true);
		window.addEventListener("resize", hide);
		return () => {
			clearTimeout(timer);
			document.removeEventListener("pointerover", over);
			document.removeEventListener("pointerdown", hide);
			document.removeEventListener("focusin", enter);
			document.removeEventListener("focusout", hide);
			document.removeEventListener("keydown", key);
			document.documentElement.removeEventListener("pointerleave", hide);
			window.removeEventListener("scroll", hide, true);
			window.removeEventListener("resize", hide);
		};
	}, []);

	// The tip is measured once it is in the document: the room below the control
	// is what decides whether it hangs under it or flips over it, and the width
	// is what decides how far a long label may go before the edge. The player bar
	// sits at the bottom of the window, so without the flip its controls answered
	// with a tip nobody could see.
	useLayoutEffect(() => {
		const el = tipRef.current;
		const shown = tip.value;
		if (!el || !shown) return;
		const box = el.getBoundingClientRect();
		const below = box.bottom > window.innerHeight - EDGE;
		const top = below ? shown.anchorTop - box.height - GAP : shown.y;
		el.style.top = `${Math.max(top, EDGE)}px`;
		const half = box.width / 2;
		const left = Math.min(
			Math.max(shown.x, EDGE + half),
			window.innerWidth - EDGE - half,
		);
		el.style.left = `${left}px`;
	});

	const shown = tip.value;
	if (!shown) return null;

	return createPortal(
		<div
			ref={tipRef}
			role="tooltip"
			style={{ left: `${shown.x}px`, top: `${shown.y}px` }}
			class="pointer-events-none fixed z-400 max-w-60 -translate-x-1/2 rounded bg-black px-2 py-1 text-center text-xs text-balance text-zinc-100 shadow-lg ring-1 ring-zinc-700"
		>
			{shown.label}
		</div>,
		document.body,
	);
}

// hover reveals the buttons that sit over a cover: they only show up once the
// pointer or the keyboard reaches them. Touch never hovers, and a long press
// already opens the item's own actions, so nothing forces them visible. Pair it
// with the corner to sit in.
export const hover =
	"absolute flex gap-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100";

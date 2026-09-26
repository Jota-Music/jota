// hover reveals the buttons that sit over a cover: they only show up once the
// pointer or the keyboard reaches them, but always on touch, where there is no
// hover to trigger them. Pair it with the corner to sit in.
export const hover =
	"pointer-coarse:opacity-100 absolute flex gap-1 opacity-0 transition group-focus-within:opacity-100 group-hover:opacity-100";

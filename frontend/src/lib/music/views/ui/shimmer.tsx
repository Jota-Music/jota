// The height the virtualised song list lays each row at, so a placeholder rows
// list and the real one share one frame.
export const ROW_PX = 64;

export function Row() {
	return (
		<div
			class="flex items-center gap-2 border-b border-zinc-900 px-3"
			style={{ height: `${ROW_PX}px` }}
		>
			<div class="shimmer size-10 shrink-0 rounded-md" />
			<div class="flex min-w-0 flex-1 flex-col gap-1.5">
				<div class="shimmer h-3.5 w-1/3 rounded" />
				<div class="shimmer h-3 w-1/5 rounded" />
			</div>
			<div class="shimmer h-3 w-8 rounded" />
		</div>
	);
}

export function Rows({ count = 12 }: { count?: number }) {
	return (
		<div class="relative min-h-0 flex-1">
			<div class="h-full overflow-y-auto rounded-lg border border-zinc-800 bg-zinc-950 pr-3.75">
				{Array.from({ length: count }, (_, index) => (
					<Row key={index} />
				))}
			</div>
		</div>
	);
}

// Tiles placeholders belong inside the same grid the real covers use, so the
// columns are the caller's and only the tile itself lives here.
export function Tiles({ count = 12 }: { count?: number }) {
	return (
		<>
			{Array.from({ length: count }, (_, index) => (
				<div key={index} class="flex flex-col gap-2 p-1.5">
					<div class="shimmer aspect-square w-full rounded-md" />
					<div class="shimmer h-5 w-3/4 rounded" />
				</div>
			))}
		</>
	);
}

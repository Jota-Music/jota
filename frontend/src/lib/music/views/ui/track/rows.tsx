export const ROW_PX = 64;

export function Row() {
	return (
		<div
			className="flex items-center gap-2 border-b border-zinc-900 px-3"
			style={{ height: `${ROW_PX}px` }}
		>
			<div className="shimmer size-10 shrink-0 rounded-md" />
			<div className="flex min-w-0 flex-1 flex-col gap-1.5">
				<div className="shimmer h-3.5 w-1/3 rounded" />
				<div className="shimmer h-3 w-1/5 rounded" />
			</div>
			<div className="shimmer h-3 w-8 rounded" />
		</div>
	);
}

export function Rows({ count = 12 }: { count?: number }) {
	return (
		<div className="relative min-h-0 flex-1">
			<div className="h-full overflow-y-auto pr-3.75 rounded-lg border border-zinc-800 bg-zinc-950">
				{Array.from({ length: count }, (_, index) => (
					<Row key={index} />
				))}
			</div>
		</div>
	);
}

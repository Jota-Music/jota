import { Disc3 } from "lucide-preact";

export function PageHeader({
	cover,
	title,
	subtitle,
}: {
	cover?: string;
	title: string;
	subtitle?: string;
}) {
	return (
		<header class="flex shrink-0 items-center gap-4">
			{cover ? (
				<img
					src={cover}
					alt=""
					class="h-16 w-16 shrink-0 rounded-md object-cover"
				/>
			) : (
				<div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-md bg-zinc-900 text-zinc-500">
					<Disc3 size={28} />
				</div>
			)}
			<div class="min-w-0">
				<h2 class="truncate text-xl font-semibold leading-tight">{title}</h2>
				{subtitle && <p class="text-sm opacity-70 mt-1">{subtitle}</p>}
			</div>
		</header>
	);
}

export default PageHeader;

import { Disc3, LayoutGrid, List } from "lucide-preact";
import { useState } from "preact/hooks";
import { Link } from "wouter-preact";

const storageKey = "cover_grid_view";

type Variant = "grid" | "compact";

function loadVariant(): Variant {
	return localStorage.getItem(storageKey) === "compact" ? "compact" : "grid";
}

export interface Item {
	id: string;
	name: string;
	cover?: string;
	subtitle?: string;
}

interface Props {
	items: Item[];
	to: (id: string) => string;
	isLoading?: boolean;
	emptyMessage?: string;
}

export function Shelf({
	items,
	to,
	isLoading = false,
	emptyMessage = "No items found.",
}: Props) {
	const [variant, setVariant] = useState<Variant>(loadVariant);

	const toggle = (next: Variant) => {
		setVariant(next);
		localStorage.setItem(storageKey, next);
	};

	return (
		<div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border border-zinc-800 bg-zinc-950">
			{isLoading ? (
				<div class="flex flex-1 items-center justify-center p-8 text-sm text-zinc-400">
					Loading...
				</div>
			) : items.length === 0 ? (
				<div class="flex flex-1 items-center justify-center p-8 text-sm text-zinc-500">
					{emptyMessage}
				</div>
			) : (
				<>
					<div class="flex shrink-0 justify-end px-3 py-2">
						<div class="flex overflow-hidden rounded-md border border-zinc-800">
							<button
								type="button"
								onClick={() => toggle("grid")}
								aria-label="Vista grilla"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center transition-colors ${
									variant === "grid"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<LayoutGrid size={14} />
							</button>
							<button
								type="button"
								onClick={() => toggle("compact")}
								aria-label="Vista compacta"
								class={`flex h-8 w-8 cursor-pointer items-center justify-center border-l border-zinc-800 transition-colors ${
									variant === "compact"
										? "bg-zinc-800 text-white"
										: "text-zinc-500 hover:text-white"
								}`}
							>
								<List size={14} />
							</button>
						</div>
					</div>

					<div class="min-h-0 flex-1 overflow-y-auto px-3 pb-3">
						{variant === "grid" ? (
							<div class="grid grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
								{items.map((item) => (
									<Link key={item.id} href={to(item.id)}>
										<div class="group flex cursor-pointer flex-col gap-2 overflow-hidden rounded-md">
											<div class="aspect-square w-full overflow-hidden rounded-md bg-zinc-900">
												{item.cover ? (
													<img
														src={item.cover}
														alt={item.name}
														loading="lazy"
														decoding="async"
														class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
													/>
												) : (
													<div class="flex h-full w-full items-center justify-center text-zinc-600">
														<Disc3 size={28} />
													</div>
												)}
											</div>
											<div class="flex flex-col">
												<h3 class="truncate text-sm font-medium text-zinc-300 transition-colors group-hover:text-white">
													{item.name}
												</h3>
												{item.subtitle && (
													<p class="text-xs text-zinc-500">{item.subtitle}</p>
												)}
											</div>
										</div>
									</Link>
								))}
							</div>
						) : (
							<ul class="flex flex-col">
								{items.map((item) => (
									<li key={item.id}>
										<Link href={to(item.id)}>
											<div class="group flex cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-zinc-900">
												<div class="h-10 w-10 shrink-0 overflow-hidden rounded bg-zinc-900">
													{item.cover ? (
														<img
															src={item.cover}
															alt={item.name}
															loading="lazy"
															decoding="async"
															class="h-full w-full object-cover transition-opacity group-hover:opacity-80"
														/>
													) : (
														<div class="flex h-full w-full items-center justify-center text-zinc-600">
															<Disc3 size={14} />
														</div>
													)}
												</div>
												<div class="flex min-w-0 flex-col">
													<h3 class="truncate text-sm text-zinc-300 group-hover:text-white">
														{item.name}
													</h3>
													{item.subtitle && (
														<p class="truncate text-xs text-zinc-500">
															{item.subtitle}
														</p>
													)}
												</div>
											</div>
										</Link>
									</li>
								))}
							</ul>
						)}
					</div>
				</>
			)}
		</div>
	);
}

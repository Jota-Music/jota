import { System } from "@wailsio/runtime";
import {
	ArrowLeft,
	ChevronDown,
	House,
	LogOut,
	Search,
	Settings,
	Users,
} from "lucide-preact";
import { useCallback, useState } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";
import { disconnectSpotify } from "@/lib/auth/views/stores/session";
import { cn } from "@/lib/shared/utils/tw";
import { WindowControlsBar } from "@/lib/shared/views/ui/components/window-controls-bar";
import { showSync } from "@/lib/sync/views/stores";

const searchTypeOptions = [
	{ value: "user", label: "User" },
	{ value: "track", label: "Track" },
	{ value: "album", label: "Album" },
	{ value: "playlist", label: "Playlist" },
	{ value: "artist", label: "Artist" },
	{ value: "youtube", label: "YouTube" },
] as const;

export function Header({
	class: _class,
	className,
	onDragStart,
}: {
	class?: string;
	className?: string;
	onDragStart?: (e: MouseEvent) => void;
}) {
	const [, setLocation] = useLocation();
	const desktop = System.IsDesktop();
	const [openSearch, setOpenSearch] = useState(false);
	const [searchDraft, setSearchDraft] = useState("");
	const [searchType, setSearchType] = useState<
		"user" | "track" | "album" | "playlist" | "artist" | "youtube"
	>("user");

	const onSearchSubmit = useCallback(
		(e: Event) => {
			e.preventDefault();
			if (!searchDraft.trim()) return;
			setLocation(
				`/search/${searchType}/${encodeURIComponent(searchDraft.trim())}`,
			);
			setSearchDraft("");
			setOpenSearch(false);
		},
		[searchDraft, searchType, setLocation],
	);

	const goBack = useCallback(() => {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			setLocation("/");
		}
	}, [setLocation]);

	function handleDragMouseDown(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (
			target.closest(
				"button, a, input, select, textarea, [role='slider'], [data-no-drag]",
			)
		)
			return;
		onDragStart?.(e);
	}

	return (
		<header
			style="--wails-draggable: drag"
			onMouseDown={handleDragMouseDown}
			class={cn(
				"sticky top-0 z-40 md:z-100 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur",
				_class,
				className,
			)}
		>
			<div class="mx-auto flex items-center justify-between text-sm text-zinc-300 h-10 px-4">
				<div class="flex items-center gap-3 pl-1">
					<button
						type="button"
						onClick={goBack}
						class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<ArrowLeft class="size-4" />
					</button>

					<Link href="/" class="text-zinc-400 hover:text-zinc-100">
						<House class="size-4" />
					</Link>
				</div>

				<div class="flex items-center gap-3">
					<button
						type="button"
						onClick={() => setOpenSearch((v) => !v)}
						class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<Search class="size-4" />
					</button>

					<button
						onClick={() => (showSync.value = !showSync.value)}
						type="button"
						title="Listen together"
						class="cursor-pointer"
					>
						<Users class="size-4 text-zinc-400 hover:text-zinc-100" />
					</button>
				</div>

				<div class="flex items-center gap-3 pr-1 h-full">
					<button
						onClick={() => void disconnectSpotify()}
						type="button"
						title="Disconnect Spotify"
						class="cursor-pointer"
					>
						<LogOut class="size-4 text-zinc-400 hover:text-zinc-100" />
					</button>

					<Link
						href="/settings"
						title="Settings"
						class="text-zinc-400 hover:text-zinc-100"
					>
						<Settings class="size-4" />
					</Link>

					{desktop && <div class="h-full w-26" aria-hidden />}
					<WindowControlsBar />
				</div>
			</div>

			<div
				class={cn(
					"mx-auto max-w-2xl overflow-hidden transition-all duration-200 px-4 md:px-0",
					openSearch ? "max-h-40 opacity-100 py-2" : "max-h-0 opacity-0 py-0",
				)}
			>
				<form
					onSubmit={onSearchSubmit}
					class="grid grid-cols-[1fr_max-content] gap-2"
				>
					<div class="flex-1 flex items-stretch h-10 rounded-lg border border-zinc-800 bg-zinc-950 overflow-hidden focus-within:border-zinc-600 focus-within:ring-1 focus-within:ring-zinc-600 transition-all">
						<div class="relative shrink-0">
							<select
								value={searchType}
								onChange={(e) =>
									setSearchType(
										(e.target as HTMLSelectElement).value as typeof searchType,
									)
								}
								class="h-full pl-3 pr-7 text-sm text-white outline-none appearance-none cursor-pointer bg-transparent border-r border-zinc-800"
							>
								{searchTypeOptions.map((opt) => (
									<option key={opt.value} value={opt.value} class="bg-zinc-950">
										{opt.label}
									</option>
								))}
							</select>
							<ChevronDown
								size={12}
								class="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none text-zinc-400"
							/>
						</div>
						<div class="relative flex-1 flex items-center">
							<input
								class="h-full w-full bg-transparent pl-4 pr-3 text-sm text-white outline-none placeholder:text-zinc-600"
								placeholder={
									searchType === "user"
										? "Spotify username..."
										: searchType === "youtube"
											? "YouTube video..."
											: `Spotify ${searchType.charAt(0).toUpperCase() + searchType.slice(1)} ID or URI...`
								}
								value={searchDraft}
								onInput={(e) =>
									setSearchDraft((e.target as HTMLInputElement).value)
								}
							/>
						</div>
					</div>

					<button
						class="h-10 w-10 md:flex-none rounded-lg px-3 py-2 text-(--binary-color) bg-(--dominant-color) hover:opacity-75 transition-opacity cursor-pointer flex items-center justify-center"
						type="submit"
						title="Search"
					>
						<Search class="size-4" />
					</button>
				</form>
			</div>
		</header>
	);
}

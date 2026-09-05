import {
	ArrowLeft,
	ArrowRight,
	ChevronDown,
	Copy,
	House,
	LockKeyhole,
	LockKeyholeOpen,
	LogOut,
	Search,
	Settings,
	Turntable,
	Undo2,
	Users,
} from "lucide-preact";
import { useCallback, useState } from "preact/hooks";
import { Link, useLocation } from "wouter-preact";
import {
	currentUser,
	disconnectSpotify,
} from "@/lib/auth/views/stores/session";
import {
	goToMyHomeRoom,
	homeRoomEnforced,
	joinRoomById,
	roomState,
} from "@/lib/shared/api/room";
import { ws } from "@/lib/shared/api/socket";
import { cn } from "@/lib/shared/utils/tw";

const searchTypeOptions = [
	{ value: "user", label: "User" },
	{ value: "track", label: "Track" },
	{ value: "album", label: "Album" },
	{ value: "playlist", label: "Playlist" },
	{ value: "artist", label: "Artist" },
] as const;

export function Header({
	class: _class,
	className,
}: {
	class?: string;
	className?: string;
}) {
	const [, setLocation] = useLocation();

	const [joinDraft, setJoinDraft] = useState("");
	const [openSettings, setOpenSettings] = useState(false);
	const [openSearch, setOpenSearch] = useState(false);
	const [searchDraft, setSearchDraft] = useState("");
	const [searchType, setSearchType] = useState<
		"user" | "track" | "album" | "playlist" | "artist"
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

	const onJoinSubmit = useCallback(
		(e: Event) => {
			e.preventDefault();
			if (!joinDraft) return;
			joinRoomById(joinDraft);
			setJoinDraft("");
			setOpenSettings(false);
		},
		[joinDraft],
	);

	const user = currentUser.value;

	const { id: room, visibility: vis, guests, youAreOwner } = roomState.value;

	const browsing = user != null && !homeRoomEnforced.value;

	const canToggleVisibility = youAreOwner || (user != null && room === user);

	const copyRoomId = useCallback(() => {
		if (!room) return;
		navigator.clipboard.writeText(`${window.location.origin}/join/${room}`);
	}, [room]);

	// ✅ BACK CORRECTO
	const goBack = useCallback(() => {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			setLocation("/");
		}
	}, [setLocation]);

	return (
		<header
			class={cn(
				"sticky top-0 z-40 border-b border-zinc-800 bg-zinc-950/80 backdrop-blur",
				_class,
				className,
			)}
		>
			{/* Top bar */}
			<div class="mx-auto flex max-w-2xl px-2 md:px-0 items-center justify-between py-2 text-sm text-zinc-300">
				<div class="flex items-center gap-3">
					{/* BACK */}
					<button
						type="button"
						onClick={goBack}
						class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<ArrowLeft class="size-4" />
					</button>

					{/* HOME */}
					<Link href="/" class="text-zinc-400 hover:text-zinc-100">
						<House class="size-4" />
					</Link>
				</div>

				<div class="flex items-center gap-3">
					<div class="flex items-center gap-2 text-xs text-zinc-500">
						<span class="flex gap-0.5 items-center justify-center">
							<Users class="size-3" />
							{guests ?? 0}
						</span>

						<button
							type="button"
							class={cn(
								"rounded p-1.5 flex items-center cursor-pointer",
								vis === "private"
									? "bg-amber-900/40 text-amber-200"
									: "bg-emerald-900/30 text-emerald-200",
							)}
							disabled={!canToggleVisibility}
							onClick={() =>
								ws.send("set-visibility", {
									visibility: vis === "private" ? "public" : "private",
								})
							}
						>
							{vis === "private" ? (
								<LockKeyhole class="size-3" />
							) : (
								<LockKeyholeOpen class="size-3" />
							)}
						</button>
					</div>

					<button
						type="button"
						onClick={() => setOpenSettings((v) => !v)}
						class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<Turntable class="size-4" />
					</button>

					<button
						type="button"
						onClick={() => setOpenSearch((v) => !v)}
						class="text-zinc-400 hover:text-zinc-100 cursor-pointer"
					>
						<Search class="size-4" />
					</button>
				</div>

				<div class="flex items-center gap-2">
					<Link href="/settings" class="hidden">
						<Settings class="size-4 text-zinc-400 hover:text-zinc-100" />
					</Link>
					<button
						onClick={() => void disconnectSpotify()}
						type="button"
						title="Disconnect Spotify"
						class="cursor-pointer"
					>
						<LogOut class="size-4 text-zinc-400 hover:text-zinc-100" />
					</button>
				</div>
			</div>

			{/* Room form */}
			<div
				class={cn(
					"mx-auto max-w-2xl overflow-hidden transition-all duration-200",
					openSettings ? "max-h-32 opacity-100 py-2" : "max-h-0 opacity-0 py-0",
				)}
			>
				<form
					onSubmit={onJoinSubmit}
					class="flex flex-col md:flex-row md:items-center gap-2 px-2 md:px-0"
				>
					<div class="flex-1 relative">
						<Users class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-zinc-500 pointer-events-none" />
						<input
							class="h-10 w-full rounded-lg border border-zinc-800 bg-zinc-950 pl-10 pr-3 text-sm text-white outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 placeholder:text-zinc-600"
							placeholder={room ?? "Room ID..."}
							value={joinDraft}
							onInput={(e) =>
								setJoinDraft((e.target as HTMLInputElement).value)
							}
						/>
					</div>

					<div class="flex items-center gap-2 w-full md:w-auto">
						<button
							class="shrink-0 h-10 w-10 rounded-lg text-(--binary-color) bg-(--dominant-color) hover:opacity-75 cursor-pointer flex items-center justify-center transition-opacity"
							type="submit"
							title="Join"
						>
							<ArrowRight class="size-4" />
						</button>

						<button
							type="button"
							class="shrink-0 h-10 w-10 rounded-lg border border-zinc-800 bg-zinc-950 text-zinc-500 hover:text-zinc-300 hover:border-zinc-600 hover:bg-zinc-900 cursor-pointer flex items-center justify-center transition-colors"
							onClick={() => goToMyHomeRoom()}
							disabled={!browsing && room === user}
							title="Mi sala"
						>
							<Undo2 class="size-4" />
						</button>

						<button
							type="button"
							class="shrink-0 h-10 w-10 rounded-lg border border-zinc-800 bg-zinc-950 text-zinc-500 hover:text-zinc-300 hover:border-zinc-600 hover:bg-zinc-900 cursor-pointer flex items-center justify-center transition-colors"
							onClick={copyRoomId}
							title="Copiar ID"
						>
							<Copy class="size-4" />
						</button>
					</div>
				</form>
			</div>

			{/* Search form */}
			<div
				class={cn(
					"mx-auto max-w-2xl overflow-hidden transition-all duration-200",
					openSearch ? "max-h-40 opacity-100 py-2" : "max-h-0 opacity-0 py-0",
				)}
			>
				<form
					onSubmit={onSearchSubmit}
					class="grid grid-cols-[1fr_max-content] gap-2 px-2 md:px-0"
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
							<ChevronDown class="absolute right-2 top-1/2 -translate-y-1/2 size-3.5 text-zinc-500 pointer-events-none" />
						</div>
						<div class="relative flex-1 flex items-center">
							<input
								class="h-full w-full bg-transparent pl-4 pr-3 text-sm text-white outline-none placeholder:text-zinc-600"
								placeholder={
									searchType === "user"
										? "Spotify username..."
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

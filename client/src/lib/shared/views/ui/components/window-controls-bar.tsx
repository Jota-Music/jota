import { Copy, Minus, X } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import {
	Quit,
	WindowIsMaximised,
	WindowMinimise,
	WindowToggleMaximise,
} from "@/wailsjs/runtime/runtime";

export function WindowControlsBar() {
	const [isMaximised, setIsMaximised] = useState(false);

	useEffect(() => {
		void WindowIsMaximised().then(setIsMaximised);
	}, []);

	const onMaximize = async () => {
		WindowToggleMaximise();
		setIsMaximised(await WindowIsMaximised());
	};

return (
			<div class="flex items-center h-full" style={{ WebkitAppRegion: "no-drag" }}>
				<button
					type="button"
					onClick={WindowMinimise}
					class="h-full px-2 text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
					title="Minimise"
					style={{ WebkitAppRegion: "no-drag" }}
				>
					<Minus class="size-3.5" />
				</button>
				<button
					type="button"
					onClick={onMaximize}
					class="h-full px-2 text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
					title={isMaximised ? "Restore" : "Maximise"}
					style={{ WebkitAppRegion: "no-drag" }}
				>
					<Copy class="size-3" />
				</button>
				<button
					type="button"
					onClick={Quit}
					class="h-full px-2 text-zinc-400 hover:text-red-400 hover:bg-red-500/10 transition-colors cursor-pointer"
					title="Close"
					style={{ WebkitAppRegion: "no-drag" }}
				>
					<X class="size-3.5" />
				</button>
			</div>
		);
}
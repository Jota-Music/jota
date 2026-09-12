import { Application, System, Window } from "@wailsio/runtime";
import { Copy, Minus, X } from "lucide-preact";
import { useEffect, useState } from "preact/hooks";

export function WindowControlsBar() {
	const isDesktop = System.IsDesktop();
	const [isMaximised, setIsMaximised] = useState(false);

	useEffect(() => {
		if (!isDesktop) return;
		void Window.IsMaximised().then(setIsMaximised);
	}, [isDesktop]);

	if (!isDesktop) return null;

	const onMaximize = async () => {
		const maximised = await Window.IsMaximised();
		if (maximised) {
			await Window.Restore();
		} else {
			await Window.Maximise();
		}
		setIsMaximised(await Window.IsMaximised());
	};

	const onQuit = async () => {
		await Application.Quit();
	};

	return (
		<div
			class="flex items-center h-full"
			style="--wails-draggable: no-drag; -webkit-app-region: no-drag;"
		>
			<button
				type="button"
				onClick={() => void Window.Minimise()}
				class="h-full px-2 text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
				title="Minimise"
			>
				<Minus class="size-3.5" />
			</button>
			<button
				type="button"
				onClick={() => void onMaximize()}
				class="h-full px-2 text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
				title={isMaximised ? "Restore" : "Maximise"}
			>
				<Copy class="size-3" />
			</button>
			<button
				type="button"
				onClick={() => void onQuit()}
				class="h-full px-2 text-zinc-400 hover:text-red-400 hover:bg-red-500/10 transition-colors cursor-pointer"
				title="Close"
			>
				<X class="size-3.5" />
			</button>
		</div>
	);
}

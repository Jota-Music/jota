import { Application, Events, System, Window } from "@wailsio/runtime";
import { Copy, Minus, Pin, X } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect, useState } from "preact/hooks";

export function WindowControlsBar() {
	const isDesktop = System.IsDesktop();
	const [isMaximised, setIsMaximised] = useState(false);
	const [isOnTop, setIsOnTop] = useState(false);

	useEffect(() => {
		if (!isDesktop) return;
		void Window.IsMaximised().then(setIsMaximised);
		const off = Events.On("window:always-on-top", (ev) =>
			setIsOnTop(Boolean(ev.data)),
		);
		return off;
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

	return createPortal(
		<div
			class="fixed top-0 right-0 z-200 flex h-10 items-center bg-zinc-950"
			style="--wails-draggable: no-drag; -webkit-app-region: no-drag;"
		>
			<button
				type="button"
				onClick={() => void Events.Emit("window:always-on-top:toggle")}
				class="flex aspect-square h-full items-center justify-center text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
				title={isOnTop ? "Desactivar siempre encima" : "Siempre encima"}
			>
				<Pin
					class={`size-3.5 transition-transform duration-150 ease-out ${
						isOnTop ? "translate-y-[2px] text-zinc-100" : "rotate-[25deg]"
					}`}
				/>
			</button>
			<button
				type="button"
				onClick={() => void Window.Minimise()}
				class="flex aspect-square h-full items-center justify-center text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
				title="Minimise"
			>
				<Minus class="size-3.5" />
			</button>
			<button
				type="button"
				onClick={() => void onMaximize()}
				class="flex aspect-square h-full items-center justify-center text-zinc-400 hover:text-zinc-100 hover:bg-white/5 transition-colors cursor-pointer"
				title={isMaximised ? "Restore" : "Maximise"}
			>
				<Copy class="size-3.5" />
			</button>
			<button
				type="button"
				onClick={() => void onQuit()}
				class="flex aspect-square h-full items-center justify-center text-zinc-400 hover:text-red-400 hover:bg-red-500/10 transition-colors cursor-pointer"
				title="Close"
			>
				<X class="size-3.5" />
			</button>
		</div>,
		document.body,
	);
}

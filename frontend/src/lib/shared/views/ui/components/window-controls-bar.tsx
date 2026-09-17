import { Application, Events, System, Window } from "@wailsio/runtime";
import { Copy, Minus, Pin, X } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect, useState } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

export function WindowControlsBar() {
	const isDesktop = System.IsDesktop();
	const [isMaximised, setIsMaximised] = useState(false);
	const [isOnTop, setIsOnTop] = useState(false);

	const buttonClass =
		"flex aspect-square h-full items-center justify-center text-zinc-400 transition-colors cursor-pointer";
	const hoverClass = "hover:text-zinc-100 hover:bg-white/5";

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
				class={cn(buttonClass, hoverClass)}
				title={isOnTop ? t("window.unpin") : t("window.pin")}
			>
				<Pin
					class={cn(
						"size-3.5 transition-transform duration-150 ease-out",
						isOnTop ? "translate-y-0.5 text-zinc-100" : "rotate-25",
					)}
				/>
			</button>
			<button
				type="button"
				onClick={() => void Window.Minimise()}
				class={cn(buttonClass, hoverClass)}
				title={t("window.minimise")}
			>
				<Minus class="size-3.5" />
			</button>
			<button
				type="button"
				onClick={() => void onMaximize()}
				class={cn(buttonClass, hoverClass)}
				title={isMaximised ? t("window.restore") : t("window.maximise")}
			>
				<Copy class="size-3.5" />
			</button>
			<button
				type="button"
				onClick={() => void onQuit()}
				class={cn(buttonClass, "hover:text-red-400 hover:bg-red-500/10")}
				title={t("window.close")}
			>
				<X class="size-3.5" />
			</button>
		</div>,
		document.body,
	);
}

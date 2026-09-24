import { useSignal } from "@preact/signals";
import { Application, Events, System, Window } from "@wailsio/runtime";
import { Copy, Minus, Pin, X } from "lucide-preact";
import { createPortal } from "preact/compat";
import { useEffect } from "preact/hooks";
import { t } from "@/lib/shared/i18n";
import { cn } from "@/lib/shared/utils/tw";

export function WindowControlsBar() {
	const isDesktop = System.IsDesktop();
	const isMaximised = useSignal(false);
	const isFullscreen = useSignal(false);
	const isOnTop = useSignal(false);

	const buttonClass =
		"flex aspect-square h-full items-center justify-center text-zinc-400 transition-colors cursor-pointer";
	const hoverClass = "hover:text-zinc-100 hover:bg-white/5";

	useEffect(() => {
		if (!isDesktop) return;
		const offMaximised = Events.On(Events.Types.Common.WindowMaximise, () => {
			isMaximised.value = true;
		});
		const offUnMaximised = Events.On(
			Events.Types.Common.WindowUnMaximise,
			() => {
				isMaximised.value = false;
			},
		);
		const offFullscreen = Events.On(
			Events.Types.Common.WindowFullscreen,
			() => {
				isFullscreen.value = true;
			},
		);
		const offUnFullscreen = Events.On(
			Events.Types.Common.WindowUnFullscreen,
			() => {
				isFullscreen.value = false;
			},
		);
		void Promise.all([Window.IsMaximised(), Window.IsFullscreen()]).then(
			([maximised, fullscreen]) => {
				isMaximised.value = maximised;
				isFullscreen.value = fullscreen;
			},
		);
		const off = Events.On("window:always-on-top", (ev) => {
			isOnTop.value = Boolean(ev.data);
		});
		// Ask for the current state instead of betting the runtime-ready
		// broadcast arrived before this subscription was in place.
		void Events.Emit("window:always-on-top:get");
		return () => {
			offMaximised();
			offUnMaximised();
			offFullscreen();
			offUnFullscreen();
			off();
		};
	}, [isDesktop]);

	if (!isDesktop) return null;

	const onMaximize = async () => {
		const [fullscreen, maximised] = await Promise.all([
			Window.IsFullscreen(),
			Window.IsMaximised(),
		]);
		if (fullscreen) {
			await Window.UnFullscreen();
		} else if (maximised) {
			await Window.Restore();
			if (await Window.IsMaximised()) await Window.UnMaximise();
		} else {
			await Window.Maximise();
		}
		isFullscreen.value = await Window.IsFullscreen();
		isMaximised.value = await Window.IsMaximised();
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
				title={isOnTop.value ? t("window.unpin") : t("window.pin")}
			>
				<Pin
					class={cn(
						"size-3.5 transition-transform duration-150 ease-out",
						isOnTop.value ? "translate-y-0.5 text-zinc-100" : "rotate-25",
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
				title={
					isFullscreen.value || isMaximised.value
						? t("window.restore")
						: t("window.maximise")
				}
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

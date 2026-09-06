import { useEffect, useRef } from "preact/hooks";
import { WindowGetPosition, WindowSetPosition, WindowIsMaximised, WindowUnmaximise } from "@/wailsjs/runtime/runtime";

export function useWindowDrag(enabled = true) {
	const dragState = useRef({ startX: 0, startY: 0, winX: 0, winY: 0, active: false });
	const handlersRef = useRef<{ down: (e: MouseEvent) => void; move: (e: MouseEvent) => void; up: () => void }>();

	useEffect(() => {
		if (!enabled) return;

		async function startDrag(e: MouseEvent) {
			if (await WindowIsMaximised()) {
				await WindowUnmaximise();
			}
			const winPos = await WindowGetPosition();
			dragState.current = {
				startX: e.screenX,
				startY: e.screenY,
				winX: winPos.x,
				winY: winPos.y,
				active: true,
			};
			document.addEventListener("mousemove", handlersRef.current!.move);
			document.addEventListener("mouseup", handlersRef.current!.up);
		}

		function onMove(e: MouseEvent) {
			const s = dragState.current;
			if (!s.active) return;
			const dx = e.screenX - s.startX;
			const dy = e.screenY - s.startY;
			WindowSetPosition(s.winX + dx, s.winY + dy);
		}

		function stopDrag() {
			dragState.current.active = false;
			document.removeEventListener("mousemove", handlersRef.current!.move);
			document.removeEventListener("mouseup", handlersRef.current!.up);
		}

		handlersRef.current = { down: startDrag, move: onMove, up: stopDrag };

		return () => {
			document.removeEventListener("mousemove", handlersRef.current!.move);
			document.removeEventListener("mouseup", handlersRef.current!.up);
		};
	}, [enabled]);

	return handlersRef.current?.down;
}

import { useEffect } from "preact/hooks";

export default function useAsyncEffect(
	callback: () => Promise<void | (() => void)>,
	deps: any[],
) {
	useEffect(() => {
		let cancelled = false;

		(async () => {
			try {
				const cleanup = await callback();
				if (typeof cleanup === "function" && !cancelled) {
					return cleanup;
				}
			} catch (e) {
				console.error("useAsyncEffect error:", e);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, deps);
}

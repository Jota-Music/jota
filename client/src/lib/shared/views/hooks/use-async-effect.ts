import { useEffect } from "preact/hooks";

export default function useAsyncEffect(
	callback: () => Promise<undefined | (() => void)>,
	deps: ReadonlyArray<unknown>,
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

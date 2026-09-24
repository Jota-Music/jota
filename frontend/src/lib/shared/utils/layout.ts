export function bottomBarHeight(): number {
	if (typeof window === "undefined") return 0;
	if (window.matchMedia("(min-width: 768px)").matches) return 0;
	return 56 + safeAreaBottom();
}

function safeAreaBottom(): number {
	const probe = document.createElement("div");
	probe.style.cssText =
		"position:fixed;visibility:hidden;height:0;padding-bottom:env(safe-area-inset-bottom)";
	document.body.appendChild(probe);
	const px = parseFloat(getComputedStyle(probe).paddingBottom) || 0;
	probe.remove();
	return px;
}

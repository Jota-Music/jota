import { ReloadWindow } from "@bindings/app";
import { Component } from "preact/compat";
import { t } from "@/lib/shared/i18n";
import { logError } from "@/lib/shared/views/stores/errors";

type Props = {
	children: preact.ComponentChildren;
};

type State = {
	crashed: boolean;
};

// A throw during render never reaches window.onerror, so without this the app
// blanks out and the user only has a log line. Reloading the webview is the
// only way back.
export class Boundary extends Component<Props, State> {
	state: State = { crashed: false };

	static getDerivedStateFromError(): State {
		return { crashed: true };
	}

	componentDidCatch(error: unknown) {
		logError(error, "render");
	}

	render() {
		if (!this.state.crashed) return <>{this.props.children}</>;

		return (
			<div className="flex h-screen flex-col items-center justify-center gap-3 bg-zinc-950 text-sm">
				<p className="text-zinc-400">{t("errors.crash")}</p>
				<button
					type="button"
					className="cursor-pointer rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 transition-colors hover:bg-zinc-800"
					onClick={() => void ReloadWindow()}
				>
					{t("common.reload")}
				</button>
			</div>
		);
	}
}

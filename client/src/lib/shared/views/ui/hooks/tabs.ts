import { effect, signal } from "@preact/signals";

/**
 * Global reactive state
 */
export const isMainTab = signal(false);

/**
 * Inter-tab communication channel
 */
const channel = new BroadcastChannel("appTabChannel");

/**
 * Initialize tab leader logic
 */
export function initTabLeader() {
    let becameMain = false;

    // Listen to other tabs
    channel.onmessage = (event) => {
        const { type, isMain: remoteIsMain } = event.data || {};

        if (type === "requestMainStatus" && isMainTab.value) {
            channel.postMessage({
                type: "mainStatus",
                isMain: true,
            });
        }

        if (type === "mainStatus") {
            if (remoteIsMain) {
                if (!becameMain) {
                    isMainTab.value = false;
                }
            }
        }
    };

    // Try to become main tab
    navigator.locks.request("appMainTab", async (lock) => {
        if (!lock) {
            isMainTab.value = false;

            channel.postMessage({
                type: "requestMainStatus",
            });

            return;
        }

        // We are the main tab
        becameMain = true;
        isMainTab.value = true;

        channel.postMessage({
            type: "mainStatus",
            isMain: true,
        });

        // Keep lock alive
        await new Promise(() => { });
    });
}

effect(() => {
    if (typeof window !== "undefined") {
        initTabLeader();
    }
});
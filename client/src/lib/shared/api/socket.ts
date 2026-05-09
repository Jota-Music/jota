import { WS_URL } from "@/lib/shared/api/env";
import { roomState, setRoomId } from "@/lib/shared/api/room";

type Listener = (data: any) => void;

type Outbound = { action: string; params?: any };

export {
    goToMyHomeRoom,
    homeRoomEnforced,
    joinRoomById,
    patchRoom,
    roomState,
    rotateGuestRoomIfForbidden,
    setRoomId
} from "@/lib/shared/api/room";
export type { RoomState, RoomVisibility } from "@/lib/shared/api/room";

class WebSocketClient {
    private socket: WebSocket | null = null;
    private listeners = new Map<string, Set<Listener>>();

    private reconnectTimeout: number | null = null;

    private outbox: Outbound[] = [];

    private prevSubscribedRoomId = roomState.peek().id;

    // flags to avoid race conditions
    private isManuallyClosed = false;
    private isRoomChanging = false;

    constructor() {
        this.connect();

        roomState.subscribe(() => {
            const id = roomState.value.id;
            if (id === this.prevSubscribedRoomId) return;
            this.prevSubscribedRoomId = id;
            this.handleRoomChange();
        });
    }

    private get room() {
        return roomState.value.id;
    }

    private buildUrl(room: string) {
        return `${WS_URL}/${room}`;
    }

    private connect() {
        const sock = new WebSocket(this.buildUrl(this.room));
        this.socket = sock;

        sock.addEventListener("open", () => {
            if (this.socket !== sock) return;
            console.log("WebSocket connected");
            this.send("join");
            this.flushOutbox();

            if (this.reconnectTimeout) {
                clearTimeout(this.reconnectTimeout);
                this.reconnectTimeout = null;
            }
        });

        sock.addEventListener("message", (event) => {
            if (this.socket !== sock) return;
            try {
                const data = JSON.parse(event.data);
                console.log("WebSocket message received:", data.action, data);

                const set = this.listeners.get(data.action);
                if (!set) {
                    console.log("No listeners for action:", data.action);
                    return;
                }

                for (const cb of set) {
                    cb(data);
                }
            } catch (err) {
                console.error("Invalid WS message", err);
            }
        });

        sock.addEventListener("close", () => {
            if (this.socket !== sock) return;
            console.log("WebSocket closed");

            this.socket = null;
            if (!this.isRoomChanging) {
                this.outbox = [];
            }

            // prevent unwanted reconnects
            if (this.isManuallyClosed || this.isRoomChanging) return;

            this.scheduleReconnect();
        });

        sock.addEventListener("error", () => {
            if (this.socket !== sock) return;
            console.log("WebSocket error");
            sock.close();
        });
    }

    private scheduleReconnect() {
        if (this.reconnectTimeout) return;

        this.reconnectTimeout = window.setTimeout(() => {
            console.log("Reconnecting WebSocket...");
            this.reconnectTimeout = null;
            this.connect();
        }, 5000);
    }

    private handleRoomChange() {
        localStorage.setItem("room", this.room);

        this.isRoomChanging = true;

        this.outbox = [];
        this.sendLeaveNow();

        this.isManuallyClosed = true;
        this.socket?.close();

        this.isManuallyClosed = false;

        this.isRoomChanging = false;

        this.connect();
    }

    setRoom(newRoom: string) {
        if (!newRoom || newRoom === this.room) return;
        setRoomId(newRoom);
    }

    private sendLeaveNow() {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
        this.socket.send(JSON.stringify({ action: "leave", params: undefined }));
    }

    private flushOutbox() {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
        const pending = this.outbox;
        this.outbox = [];
        for (const msg of pending) {
            this.socket.send(JSON.stringify(msg));
        }
    }

    send(action: string, params?: any) {
        if (action === "leave") {
            if (this.socket?.readyState === WebSocket.OPEN) {
                this.socket.send(JSON.stringify({ action, params }));
            }
            return;
        }

        const msg: Outbound = { action, params };
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            this.outbox.push(msg);
            return;
        }
        this.socket.send(JSON.stringify(msg));
    }

    on(action: string, cb: Listener) {
        if (!this.listeners.has(action)) {
            this.listeners.set(action, new Set());
        }

        this.listeners.get(action)!.add(cb);
    }

    off(action: string, cb: Listener) {
        const set = this.listeners.get(action);
        if (!set) return;

        set.delete(cb);

        if (set.size === 0) {
            this.listeners.delete(action);
        }
    }

    close() {
        this.send("leave");
        this.isManuallyClosed = true;
        this.socket?.close();
    }
}

export const ws = new WebSocketClient();

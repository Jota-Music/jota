const API_URL = import.meta.env.VITE_API_URL;

if (!API_URL) {
    throw new Error("VITE_API_URL is not set");
}

const WS_URL = import.meta.env.VITE_WS_URL;

if (!WS_URL) {
    throw new Error("VITE_WS_URL is not set");
}

export { API_URL, WS_URL };



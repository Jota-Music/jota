import "@/lib/music/views/stores/socket";
import "@/lib/shared/api/socket";
import "@/lib/shared/views/stores/error-listeners";
import "@/lib/shared/views/ui/hooks/tabs";
import { render } from "preact";
import { syncAuth } from "@/lib/auth/views/stores/session";
import Router from "@/lib/shared/views/ui/router";
import "./style.tw.css";

void syncAuth();

const appRoot = document.getElementById("app");
if (!appRoot) throw new Error("Missing #app root");
render(<Router />, appRoot);

import "@/lib/shared/views/ui/hooks/tabs";
import { render } from "preact";
import Router from "@/lib/shared/views/ui/router";
import "@wailsio/runtime";
import "./style.tw.css";

const appRoot = document.getElementById("app");
if (!appRoot) throw new Error("Missing #app root");
render(<Router />, appRoot);

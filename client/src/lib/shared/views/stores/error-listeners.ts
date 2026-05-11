import { ws } from "@/lib/shared/api/socket";
import { addError } from "@/lib/shared/views/stores/errors";

// Visibility error
ws.on("visibility-error", (msg) => {
  console.log("Visibility error received:", msg);
  addError(String(msg.errors || "Error al cambiar la visibilidad de la sala"));
});

// Generic error
ws.on("error", (msg) => {
  console.log("Error received:", msg);
  if (msg.errors) {
    addError(String(msg.errors));
  }
});

/// <reference types="node" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

const port = Number(process.env.WAILS_VITE_PORT) || 9245;

// Bind IPv4 explicitly. Wails' asset proxy dials tcp4 127.0.0.1; Vite's
// default "localhost" often listens on ::1 only → blank WebView.
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port,
    strictPort: true,
    hmr: {
      protocol: "ws",
      host: "127.0.0.1",
      port,
      clientPort: port,
    },
  },
  plugins: [react(), tailwindcss(), wails("./bindings")],
});

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import viteTsconfigPaths from "vite-tsconfig-paths";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), viteTsconfigPaths()],
  build: {
    outDir: "./build"
  },
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8089",
        changeOrigin: true,
        secure: false
      }
    }
  }
});

import tailwindcss from "@tailwindcss/vite";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";
import path from "path";

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  resolve: {
    alias: {
      $lib: path.resolve("./src/lib"),
      components: path.resolve("./src/lib/components"),
      utils: path.resolve("./src/lib/utils"),
      hooks: path.resolve("./src/lib/hooks"),
      ui: path.resolve("./src/lib/components/ui"),
    },
  },
  server: { host: true },
});

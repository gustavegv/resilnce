// prettier.config.cjs

module.exports = {
  // 🔌 Load the plugin that tells Prettier how to format .svelte files
  plugins: ["prettier-plugin-svelte"],

  // 🧹 Use single quotes instead of double quotes in JS/TS/HTML
  singleQuote: true,

  // ✅ Add semicolons at the end of statements
  semi: true,

  // 📏 Wrap lines after 100 characters (keeps code readable)
  printWidth: 100,

  // 🧩 Sort <script>, HTML markup, exported props, and <style> in this order
  svelteSortOrder: "scripts-markup-options-styles",

  // 🚨 Enforce strict, consistent formatting for Svelte files
  svelteStrictMode: true,

  // ✂️ Use shorthand syntax when possible (e.g., `prop={prop}` → `prop`)
  svelteAllowShorthand: true,
};

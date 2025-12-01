/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        // Custom colors for node types
        "node-http": "#dbeafe", // blue-100
        "node-llm": "#f3e8ff", // purple-100
        "node-transform": "#fed7aa", // orange-100
        "node-conditional": "#dcfce7", // green-100
        "node-merge": "#fce7f3", // pink-100
      },
      animation: {
        "pulse-slow": "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite",
      },
    },
  },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")],
};

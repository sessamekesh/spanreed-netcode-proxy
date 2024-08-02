import React from "react";
import { createRoot } from "react-dom/client";
import { BenchmarkPage } from "./BenchmarkPage";

function bootstrapApp() {
  const container = document.getElementById("app");
  if (!container) return;
  const root = createRoot(container);
  root.render(
    <React.StrictMode>
      <BenchmarkPage />
    </React.StrictMode>
  );
}

bootstrapApp();

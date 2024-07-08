import React from "react";
import { createRoot } from "react-dom/client";
import { DemoPage } from "./DemoPage";

function bootstrapApp() {
  const container = document.getElementById("app");
  if (!container) return;
  const root = createRoot(container);
  root.render(
    <React.StrictMode>
      <DemoPage />
    </React.StrictMode>,
  );
}

bootstrapApp();

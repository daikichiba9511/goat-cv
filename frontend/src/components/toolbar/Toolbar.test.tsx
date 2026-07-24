// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { ImageMeta } from "../../types";
import Toolbar from "./Toolbar";

const escalatedImage: ImageMeta = {
  id: "image-1",
  project_id: "project-1",
  filename: "sample.png",
  original_width: 640,
  original_height: 480,
  width: 640,
  height: 480,
  rotation: 0,
  flip_h: false,
  flip_v: false,
  status: "pending",
  escalated: true,
  uploaded_at: "2026-07-24T00:00:00Z",
};

afterEach(cleanup);

it("keeps viewing tools available while escalation disables Graph and transform controls", () => {
  render(
    <MemoryRouter>
      <Toolbar
        activeTool="select"
        onToolChange={vi.fn()}
        onSave={vi.fn()}
        dirty
        saving={false}
        saveError={null}
        projectName="Vision dataset"
        image={escalatedImage}
        graphEditable={false}
        transformEditable={false}
        onRotate={vi.fn()}
        onFlipH={vi.fn()}
        onFlipV={vi.fn()}
      />
    </MemoryRouter>,
  );

  expect((screen.getByRole("button", { name: "Select" }) as HTMLButtonElement).disabled)
    .toBe(false);
  expect((screen.getByRole("button", { name: "Pan" }) as HTMLButtonElement).disabled)
    .toBe(false);
  for (const controlName of [
    "BBox",
    "Polygon",
    "Edge",
    "Rotate 90 degrees",
    "Flip horizontally",
    "Flip vertically",
    "Save",
  ]) {
    for (const control of screen.getAllByRole("button", { name: controlName })) {
      expect((control as HTMLButtonElement).disabled).toBe(true);
    }
  }
});

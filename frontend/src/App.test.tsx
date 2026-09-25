import { render, screen } from "@testing-library/react";
import { App } from "./App";

test("shows the project name as the page heading", () => {
  render(<App />);
  expect(
    screen.getByRole("heading", { level: 1, name: "Kurswechsel" }),
  ).toBeInTheDocument();
});

test("renders inside the main landmark", () => {
  render(<App />);
  expect(screen.getByRole("main")).toBeInTheDocument();
});

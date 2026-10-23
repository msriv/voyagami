import { render, screen } from "@testing-library/react-native";

import "@/i18n";
import { ThemeProvider } from "@/theme";

import Gallery from "../app/gallery";

jest.mock("expo-router", () => ({ Redirect: () => null }));

describe("gallery", () => {
  it("shows every component group in dev", async () => {
    await render(
      <ThemeProvider>
        <Gallery />
      </ThemeProvider>,
    );
    for (const name of ["Buttons", "Card and chips", "Inputs", "Sheet and toast", "Motifs"]) {
      expect(screen.getByRole("header", { name })).toBeTruthy();
    }
  });
});

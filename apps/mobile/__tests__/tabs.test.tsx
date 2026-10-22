import { renderRouter, screen } from "expo-router/testing-library";

import RootLayout from "../app/_layout";
import TabsLayout from "../app/(tabs)/_layout";
import Explore from "../app/(tabs)/explore";
import Home from "../app/(tabs)/index";
import Profile from "../app/(tabs)/profile";

jest.mock("expo-font", () => ({ useFonts: () => [true, null] }));
jest.mock("expo-splash-screen", () => ({
  preventAutoHideAsync: jest.fn(() => Promise.resolve()),
  hideAsync: jest.fn(() => Promise.resolve()),
}));

describe("tab shell", () => {
  it("shows Home, Explore and Profile tabs", async () => {
    await renderRouter({
      _layout: RootLayout,
      "(tabs)/_layout": TabsLayout,
      "(tabs)/index": Home,
      "(tabs)/explore": Explore,
      "(tabs)/profile": Profile,
    });
    expect(screen.getByRole("button", { name: "Home" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Explore" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Profile" })).toBeTruthy();
  });
});

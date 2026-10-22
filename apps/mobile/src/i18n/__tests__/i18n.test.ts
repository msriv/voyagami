import { i18n } from "..";

describe("i18n", () => {
  it("returns English strings and falls back to English", async () => {
    await i18n.changeLanguage("fr");
    expect(i18n.t("tabs.home")).toBe("Home");
  });
});

import { render, screen } from "@testing-library/react-native";
import { StyleSheet, View } from "react-native";

import { ThemeProvider } from "@/theme";

import { BrushCircle, Postmark, Seal, WashiTape, WavePattern } from "..";

const wrap = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe("motifs", () => {
  it("renders every motif", async () => {
    await wrap(
      <View>
        <Seal />
        <WavePattern />
        <BrushCircle />
        <WashiTape />
        <Postmark city="Hanoi" date="3 Nov" />
      </View>,
    );
    expect(screen.toJSON()).toBeTruthy();
  });

  it("labels the seal only when asked", async () => {
    await wrap(<Seal accessibilityLabel="Voyagami seal" />);
    expect(screen.getByRole("image", { name: "Voyagami seal" })).toBeTruthy();
  });

  it("never draws the seal under 24 points", async () => {
    await wrap(<Seal size={10} accessibilityLabel="Seal" />);
    expect(screen.getByRole("image", { name: "Seal" }).props.width).toBe(24);
  });

  it("reads the postmark as one phrase", async () => {
    await wrap(<Postmark city="Hanoi" date="3 Nov" />);
    expect(screen.getByLabelText("Hanoi, 3 Nov")).toBeTruthy();
  });

  it("hides washi tape from screen readers and tilts it", async () => {
    const { toJSON } = await wrap(<WashiTape />);
    const tree = toJSON() as unknown as {
      props: { style: object; accessibilityElementsHidden: boolean };
    };
    expect(tree.props.accessibilityElementsHidden).toBe(true);
    expect((StyleSheet.flatten(tree.props.style) as { transform: unknown }).transform).toEqual([
      { rotate: "-8deg" },
    ]);
  });
});

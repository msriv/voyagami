import { fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react-native";
import { AccessibilityInfo, StyleSheet, View } from "react-native";

import { ThemeProvider } from "@/theme";

import { Button, IconButton, Text, useReducedMotion } from "..";
import { pressFeedback } from "../buttonStyles";

const wrap = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe("Button", () => {
  it("is a button with a label and a 44 point minimum", async () => {
    const onPress = jest.fn();
    await wrap(<Button label="Save trip" onPress={onPress} />);
    const button = screen.getByRole("button", { name: "Save trip" });
    const style = StyleSheet.flatten(button.props.style);
    expect(style.minHeight).toBeGreaterThanOrEqual(44);
    expect(style.minWidth).toBeGreaterThanOrEqual(44);
    expect(button.props.accessibilityState).toMatchObject({ disabled: false, busy: false });
    await fireEvent.press(button);
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("can say more to screen readers than it shows", async () => {
    await wrap(<Button label="Done" accessibilityLabel="Save trip and close" />);
    expect(screen.getByRole("button", { name: "Save trip and close" })).toBeTruthy();
  });

  it("is disabled and does not fire when disabled", async () => {
    const onPress = jest.fn();
    await wrap(<Button label="Save" disabled onPress={onPress} />);
    const button = screen.getByRole("button", { name: "Save" });
    expect(button.props.accessibilityState).toMatchObject({ disabled: true });
    await fireEvent.press(button);
    expect(onPress).not.toHaveBeenCalled();
  });

  it("is busy and ignores presses while loading", async () => {
    const onPress = jest.fn();
    await wrap(<Button label="Save" loading onPress={onPress} />);
    const button = screen.getByRole("button", { name: "Save" });
    expect(button.props.accessibilityState).toMatchObject({ busy: true, disabled: true });
    await fireEvent.press(button);
    expect(onPress).not.toHaveBeenCalled();
  });

  it("uses ink text on the vermilion accent, never white", async () => {
    await wrap(<Button label="Go" variant="accent" />);
    const text = screen.getByText("Go");
    expect(StyleSheet.flatten(text.props.style).color).toBe("#14161B");
  });
});

describe("IconButton", () => {
  it("is a 44 point button with the given label", async () => {
    const onPress = jest.fn();
    await wrap(
      <IconButton accessibilityLabel="Close" onPress={onPress}>
        <View />
      </IconButton>,
    );
    const button = screen.getByRole("button", { name: "Close" });
    const style = StyleSheet.flatten(button.props.style);
    expect(style.width).toBeGreaterThanOrEqual(44);
    expect(style.height).toBeGreaterThanOrEqual(44);
    await fireEvent.press(button);
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("reports disabled state", async () => {
    await wrap(
      <IconButton accessibilityLabel="Close" disabled>
        <View />
      </IconButton>,
    );
    expect(screen.getByRole("button", { name: "Close" }).props.accessibilityState).toMatchObject({
      disabled: true,
    });
  });
});

describe("Text", () => {
  it("scales with the system text size", async () => {
    await wrap(<Text>Hello</Text>);
    const node = screen.getByText("Hello");
    expect(node.props.allowFontScaling).not.toBe(false);
    expect(node.props.maxFontSizeMultiplier).toBeUndefined();
  });

  it("marks headings for screen readers", async () => {
    await wrap(
      <Text variant="display" heading>
        Trips
      </Text>,
    );
    expect(screen.getByRole("header", { name: "Trips" })).toBeTruthy();
  });

  it("uses the Brand fonts per variant", async () => {
    await wrap(
      <>
        <Text variant="display">A</Text>
        <Text variant="body">B</Text>
        <Text variant="accent">C</Text>
        <Text variant="handwriting">D</Text>
      </>,
    );
    const family = (t: string) => StyleSheet.flatten(screen.getByText(t).props.style).fontFamily;
    expect(family("A")).toBe("BricolageGrotesque_700Bold");
    expect(family("B")).toBe("Manrope_400Regular");
    expect(family("C")).toBe("InstrumentSerif_400Regular_Italic");
    expect(family("D")).toBe("Caveat_600SemiBold");
  });
});

describe("reduce motion", () => {
  it("scales a pressed control only when reduce motion is off", () => {
    expect(pressFeedback(true, false).transform).toEqual([{ scale: 0.97 }]);
    expect(pressFeedback(true, true).transform).toEqual([]);
    expect(pressFeedback(false, false).transform).toEqual([]);
    expect(pressFeedback(true, true).opacity).toBeLessThan(1);
  });

  it("reads the system setting", async () => {
    jest.spyOn(AccessibilityInfo, "isReduceMotionEnabled").mockResolvedValue(true);
    const { result } = await renderHook(() => useReducedMotion());
    await waitFor(() => expect(result.current).toBe(true));
  });
});

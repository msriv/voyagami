import { act, fireEvent, render, screen } from "@testing-library/react-native";
import { StyleSheet, Text as RNText } from "react-native";

import { ThemeProvider } from "@/theme";

import { Card, Chip, Input, Sheet, Toast } from "..";

const wrap = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe("Card", () => {
  it("has radius 24", async () => {
    await wrap(
      <Card>
        <RNText>Inside</RNText>
      </Card>,
    );
    const card = screen.getByText("Inside").parent as NonNullable<
      ReturnType<typeof screen.getByText>["parent"]
    >;
    expect(StyleSheet.flatten(card.props.style).borderRadius).toBe(24);
  });
});

describe("Chip", () => {
  it("is a 44 point button that reports selected and toggles", async () => {
    const onPress = jest.fn();
    await wrap(<Chip label="Food" selected={false} onPress={onPress} />);
    const chip = screen.getByRole("button", { name: "Food" });
    expect(StyleSheet.flatten(chip.props.style).minHeight).toBeGreaterThanOrEqual(44);
    expect(chip.props.accessibilityState).toMatchObject({ selected: false, disabled: false });
    await fireEvent.press(chip);
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("reports selected state", async () => {
    await wrap(<Chip label="Food" selected />);
    expect(screen.getByRole("button", { name: "Food" }).props.accessibilityState).toMatchObject({
      selected: true,
    });
  });

  it("does not fire when disabled", async () => {
    const onPress = jest.fn();
    await wrap(<Chip label="Food" selected={false} disabled onPress={onPress} />);
    await fireEvent.press(screen.getByRole("button", { name: "Food" }));
    expect(onPress).not.toHaveBeenCalled();
  });
});

describe("Input", () => {
  it("shows a visible label and labels the field", async () => {
    await wrap(<Input label="Trip name" value="" onChangeText={() => {}} />);
    expect(screen.getByText("Trip name")).toBeTruthy();
    expect(screen.getByLabelText("Trip name")).toBeTruthy();
  });

  it("shows the error, marks the field invalid and uses a live region", async () => {
    await wrap(<Input label="Trip name" error="Enter a name" value="" onChangeText={() => {}} />);
    const error = screen.getByText("Enter a name");
    expect(error.props.accessibilityLiveRegion).toBe("polite");
    expect(screen.getByLabelText("Trip name").props["aria-invalid"]).toBe(true);
  });

  it("is at least 44 high and passes typing through", async () => {
    const onChangeText = jest.fn();
    await wrap(<Input label="Trip name" value="" onChangeText={onChangeText} />);
    const field = screen.getByLabelText("Trip name");
    expect(StyleSheet.flatten(field.props.style).minHeight).toBeGreaterThanOrEqual(44);
    await fireEvent.changeText(field, "Hanoi");
    expect(onChangeText).toHaveBeenCalledWith("Hanoi");
  });
});

describe("Sheet", () => {
  const sheet = (visible: boolean, onClose = jest.fn()) => (
    <Sheet
      visible={visible}
      onClose={onClose}
      title="Filters"
      closeLabel="Close"
      closeIcon={<RNText>x</RNText>}
    >
      <RNText>Body</RNText>
    </Sheet>
  );

  it("renders a heading and content when visible, nothing when hidden", async () => {
    const view = await wrap(sheet(true));
    expect(screen.getByRole("header", { name: "Filters" })).toBeTruthy();
    expect(screen.getByText("Body")).toBeTruthy();
    await view.rerender(<ThemeProvider>{sheet(false)}</ThemeProvider>);
    expect(screen.queryByText("Body")).toBeNull();
  });

  it("closes from the close button", async () => {
    const onClose = jest.fn();
    await wrap(sheet(true, onClose));
    await fireEvent.press(
      screen.getAllByRole("button", { name: "Close" }).at(-1) as ReturnType<
        typeof screen.getByText
      >,
    );
    expect(onClose).toHaveBeenCalled();
  });
});

describe("Toast", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("is an alert with a dismiss button", async () => {
    const onDismiss = jest.fn();
    await wrap(
      <Toast
        message="Saved"
        onDismiss={onDismiss}
        dismissLabel="Dismiss"
        dismissIcon={<RNText>x</RNText>}
      />,
    );
    expect(screen.getByRole("alert")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Dismiss" }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("dismisses itself after the duration", async () => {
    const onDismiss = jest.fn();
    await wrap(
      <Toast
        message="Saved"
        onDismiss={onDismiss}
        dismissLabel="Dismiss"
        dismissIcon={<RNText>x</RNText>}
        durationMs={3000}
      />,
    );
    await act(async () => {
      jest.advanceTimersByTime(2999);
    });
    expect(onDismiss).not.toHaveBeenCalled();
    await act(async () => {
      jest.advanceTimersByTime(1);
    });
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("stays until dismissed when duration is 0", async () => {
    const onDismiss = jest.fn();
    await wrap(
      <Toast
        message="Saved"
        onDismiss={onDismiss}
        dismissLabel="Dismiss"
        dismissIcon={<RNText>x</RNText>}
        durationMs={0}
      />,
    );
    await act(async () => {
      jest.advanceTimersByTime(60000);
    });
    expect(onDismiss).not.toHaveBeenCalled();
  });
});

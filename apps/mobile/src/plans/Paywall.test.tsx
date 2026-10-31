import { fireEvent, render, screen } from "@testing-library/react-native";

import "@/i18n";
import { ThemeProvider } from "@/theme";

import { PaywallProvider, usePaywall } from "./Paywall";
import { PlanPricesProvider } from "./prices";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ useRouter: () => ({ push: mockPush }) }));

function Trigger({ feature }: { feature: string }) {
  const { show } = usePaywall();
  const { Pressable, Text } = jest.requireActual("react-native");
  return (
    <Pressable accessibilityRole="button" accessibilityLabel="open" onPress={() => show(feature)}>
      <Text>{"open"}</Text>
    </Pressable>
  );
}

const setup = (feature: string, week?: string) =>
  render(
    <ThemeProvider>
      <PlanPricesProvider value={week ? { pass: { week } } : {}}>
        <PaywallProvider>
          <Trigger feature={feature} />
        </PaywallProvider>
      </PlanPricesProvider>
    </ThemeProvider>,
  );

describe("Paywall", () => {
  beforeEach(() => mockPush.mockClear());

  it("opens with the best order words and closes with Not now", async () => {
    await setup("bestOrder");
    await fireEvent.press(screen.getByLabelText("open"));
    expect(screen.getByText("Find the best order again")).toBeTruthy();
    await fireEvent.press(screen.getAllByRole("button", { name: "Not now" }).at(-1) as never);
    expect(screen.queryByText("Find the best order again")).toBeNull();
    expect(mockPush).not.toHaveBeenCalled();
  });

  it("opens with the second currency words and goes to Plans", async () => {
    await setup("multiCurrency");
    await fireEvent.press(screen.getByLabelText("open"));
    expect(screen.getByText("Add a second currency?")).toBeTruthy();
    await fireEvent.press(screen.getByRole("button", { name: "Try Plus free for 7 days" }));
    expect(mockPush).toHaveBeenCalledWith("/plans");
  });

  it("shows the store's pass price only when there is one, and general words for other features", async () => {
    await setup("files", "€4.99");
    await fireEvent.press(screen.getByLabelText("open"));
    expect(screen.getByText("This is a Plus feature")).toBeTruthy();
    expect(screen.getByText("Or a week Trip Pass for €4.99")).toBeTruthy();
  });
});

import { act, render, screen } from "@testing-library/react-native";

import { ThemeProvider } from "@/theme";

import { enqueue, flush } from "./queue";
import { SyncStatus } from "./SyncStatus";
import { deriveSync, markReachable } from "./syncStatus";

const wrap = () =>
  render(
    <ThemeProvider>
      <SyncStatus />
    </ThemeProvider>,
  );

const expense = (id: string) => ({
  op: "createExpense" as const,
  tripId: "t1",
  id,
  body: { id } as never,
});

afterEach(() => markReachable(true));

describe("deriveSync", () => {
  it("puts a refused change first, then no signal, then waiting, then all fine", () => {
    expect(deriveSync(0, 0, false).state).toBe("synced");
    expect(deriveSync(2, 0, false).state).toBe("syncing");
    expect(deriveSync(2, 0, true).state).toBe("offline");
    expect(deriveSync(2, 1, true).state).toBe("attention");
  });
});

describe("SyncStatus", () => {
  it("says everything is saved when nothing waits", async () => {
    await wrap();
    expect(await screen.findByLabelText("Everything is saved")).toBeTruthy();
  });

  it("counts changes waiting, and says so when there is no signal", async () => {
    await wrap();
    await act(async () => {
      await enqueue(expense("a"));
      await enqueue(expense("b"));
    });
    expect(await screen.findByLabelText("Sending 2 changes")).toBeTruthy();
    await act(async () => markReachable(false));
    expect(await screen.findByLabelText("No signal. 2 changes saved on your phone")).toBeTruthy();
    expect(screen.getByText(/Keep going/)).toBeTruthy();
    await act(async () => markReachable(true));
    expect(await screen.findByLabelText("Sending 2 changes")).toBeTruthy();
  });

  it("asks for attention when the server refuses a change", async () => {
    await wrap();
    await act(async () => {
      await enqueue(expense("a"));
      await flush(async () => ({ reject: "No" }));
    });
    expect(await screen.findByLabelText("1 change needs attention")).toBeTruthy();
  });

  it("is announced politely as it changes", async () => {
    await wrap();
    const status = await screen.findByLabelText("Everything is saved");
    expect(status.props.accessibilityLiveRegion).toBe("polite");
  });
});

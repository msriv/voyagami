import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";

import { ThemeProvider } from "@/theme";

import { enqueue, pendingWrites } from "./queue";
import { SavedOnPhone } from "./SavedOnPhone";
import { localStore } from "./store";

it("shows what is kept and clears it without losing changes waiting to send", async () => {
  await localStore().put("trips", [{ id: "t1", title: "x".repeat(3000) }]);
  await enqueue({
    op: "createExpense",
    tripId: "t1",
    id: "a",
    body: { id: "a" } as never,
  });
  await render(
    <ThemeProvider>
      <SavedOnPhone />
    </ThemeProvider>,
  );
  expect(await screen.findByText(/Trips and plans kept for offline use: [3-9] kB/i)).toBeTruthy();
  await fireEvent.press(screen.getByRole("button", { name: "Clear saved copies" }));
  await waitFor(() => expect(screen.getByText(/offline use: [12] kB/i)).toBeTruthy());
  expect(await localStore().get("trips")).toBeUndefined();
  expect(await pendingWrites()).toHaveLength(1);
});

import { createReporter, redact, type LogEntry } from "../remoteLog";

function setup(send: (e: LogEntry) => Promise<unknown> = () => Promise.resolve()) {
  const sent: LogEntry[] = [];
  const reporter = createReporter({
    getDeviceId: () => Promise.resolve("d-1"),
    appVersion: "1.2.3",
    send: (e) => {
      sent.push(e);
      return send(e);
    },
  });
  return { reporter, sent };
}

const flush = () => new Promise((r) => setTimeout(r, 0));

describe("remote log", () => {
  it("sends the device, version, route and the latest breadcrumbs with an error", async () => {
    const { reporter, sent } = setup();
    reporter.setRoute("/onboarding/style");
    reporter.breadcrumb("PATCH /v1/me 200");
    reporter.log("error", "boom", "stack here");
    await flush();
    expect(sent).toHaveLength(1);
    expect(sent[0]).toMatchObject({
      deviceId: "d-1",
      appVersion: "1.2.3",
      level: "error",
      message: "boom",
      stack: "stack here",
      route: "/onboarding/style",
    });
    expect(sent[0]?.breadcrumbs?.at(-1)).toMatch(/PATCH \/v1\/me 200$/);
  });

  it("keeps only the last 30 breadcrumbs", async () => {
    const { reporter, sent } = setup();
    for (let i = 0; i < 40; i++) reporter.breadcrumb(`step ${i}`);
    reporter.log("info", "x");
    await flush();
    expect(sent[0]?.breadcrumbs).toHaveLength(30);
  });

  it("never throws when sending fails", async () => {
    const { reporter } = setup(() => Promise.reject(new Error("offline")));
    expect(() => reporter.log("error", "x")).not.toThrow();
    await flush();
  });

  it("strips phone numbers", () => {
    expect(redact("code sent to +91 98765 43210 ok")).toBe("code sent to [number] ok");
  });
});

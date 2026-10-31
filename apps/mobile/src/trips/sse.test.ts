import { parseSse } from "./sse";

describe("parseSse", () => {
  it("returns whole events and keeps a half-received one for later", () => {
    const first = parseSse(
      ': open\n\nid: 1\nevent: change\ndata: {"kind":"stops"}\n\nid: 2\nevent: ch',
    );
    expect(first.events).toEqual([{ event: "change", data: '{"kind":"stops"}' }]);
    const second = parseSse(first.rest + 'ange\ndata: {"kind":"tasks"}\n\n');
    expect(second.events).toEqual([{ event: "change", data: '{"kind":"tasks"}' }]);
    expect(second.rest).toBe("");
  });

  it("reads a resync and ignores keep-alives", () => {
    expect(parseSse("id: 1\nevent: resync\ndata: {}\n\n: keep-alive\n\n").events).toEqual([
      { event: "resync", data: "{}" },
    ]);
  });
});

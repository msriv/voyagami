import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

interface IndexField {
  fieldPath: string;
  order?: "ASCENDING" | "DESCENDING";
  arrayConfig?: "CONTAINS";
}

interface CompositeIndex {
  collectionGroup: string;
  queryScope: "COLLECTION" | "COLLECTION_GROUP";
  fields: IndexField[];
}

interface FieldOverride {
  collectionGroup: string;
  fieldPath: string;
  ttl?: boolean;
  indexes: { order?: string; arrayConfig?: string; queryScope: string }[];
}

interface IndexesFile {
  indexes: CompositeIndex[];
  fieldOverrides: FieldOverride[];
}

const file = JSON.parse(
  readFileSync(fileURLToPath(new URL("../../../firestore.indexes.json", import.meta.url)), "utf8"),
) as IndexesFile;

describe("firestore.indexes.json", () => {
  it("only lists composite indexes, each with at least two fields", () => {
    for (const index of file.indexes) {
      expect(index.fields.length, index.collectionGroup).toBeGreaterThanOrEqual(2);
    }
  });

  it("has no duplicate composite index", () => {
    const keys = file.indexes.map((i) =>
      JSON.stringify([i.collectionGroup, i.queryScope, i.fields]),
    );
    expect(new Set(keys).size).toBe(keys.length);
  });

  it("indexes trips by member for the Trips list", () => {
    const trips = file.indexes.find((i) => i.collectionGroup === "trips");
    expect(trips?.fields[0]).toEqual({ fieldPath: "memberUids", arrayConfig: "CONTAINS" });
  });

  it("lets a collection group query find members by uid", () => {
    const override = file.fieldOverrides.find(
      (o) => o.collectionGroup === "members" && o.fieldPath === "uid",
    );
    expect(override?.indexes.some((i) => i.queryScope === "COLLECTION_GROUP")).toBe(true);
  });

  it("expires presence documents with a TTL on endsAt", () => {
    const override = file.fieldOverrides.find(
      (o) => o.collectionGroup === "presence" && o.fieldPath === "endsAt",
    );
    expect(override?.ttl).toBe(true);
  });
});

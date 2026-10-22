import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import {
  assertFails,
  initializeTestEnvironment,
  type RulesTestEnvironment,
} from "@firebase/rules-unit-testing";
import {
  collection,
  collectionGroup,
  deleteDoc,
  doc,
  getDoc,
  getDocs,
  query,
  setDoc,
  updateDoc,
  where,
  type Firestore,
} from "firebase/firestore";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

const rulesPath = fileURLToPath(new URL("../../../firestore.rules", import.meta.url));

// Every document path in docs/data-model.md, one example each. Add a path here
// when a collection is added to the data model.
const documentPaths = [
  "users/u1",
  "users/u1/devices/d1",
  "users/u1/stamps/s1",
  "phoneIndex/h1",
  "sessions/s1",
  "entitlements/u1",
  "invites/i1",
  "places/p1",
  "trips/t1",
  "trips/t1/members/u1",
  "trips/t1/days/2026-11-21",
  "trips/t1/stops/s1",
  "trips/t1/saved/s1",
  "trips/t1/bookings/b1",
  "trips/t1/files/f1",
  "trips/t1/polls/p1",
  "trips/t1/polls/p1/votes/u1",
  "trips/t1/tasks/k1",
  "trips/t1/comments/c1",
  "trips/t1/meetPoints/m1",
  "trips/t1/presence/u1",
  "trips/t1/expenses/e1",
  "trips/t1/settlements/s1",
  "trips/t1/journal/2026-11-21_u1",
  "trips/t1/media/m1",
  "trips/t1/passes/p1",
  "trips/t1/activity/a1",
  // Not in the data model: anything else is denied too.
  "somethingNew/x1",
  "trips/t1/notInTheModel/n1",
];

const collectionPath = (docPath: string): string => docPath.split("/").slice(0, -1).join("/");

let env: RulesTestEnvironment;

beforeAll(async () => {
  env = await initializeTestEnvironment({
    projectId: "demo-voyagami",
    firestore: { rules: readFileSync(rulesPath, "utf8") },
  });
  await env.clearFirestore();
  await env.withSecurityRulesDisabled(async (ctx) => {
    const db = ctx.firestore();
    for (const path of documentPaths) {
      await setDoc(doc(db, path), { seeded: true });
    }
    // u1 is the owner of t1, so membership alone must not open anything.
    await setDoc(doc(db, "trips/t1/members/u1"), { uid: "u1", role: "owner", status: "active" });
    await setDoc(doc(db, "trips/t1"), { ownerUid: "u1", memberUids: ["u1"] });
  });
});

afterAll(async () => {
  await env.cleanup();
});

const clients: [string, () => Firestore][] = [
  ["a signed-out client", () => env.unauthenticatedContext().firestore() as unknown as Firestore],
  [
    "a signed-in stranger",
    () => env.authenticatedContext("stranger").firestore() as unknown as Firestore,
  ],
  [
    "the trip's owner",
    () => env.authenticatedContext("u1", { role: "owner" }).firestore() as unknown as Firestore,
  ],
];

describe("setup", () => {
  it("has the seeded data, so the denials below are real", async () => {
    await env.withSecurityRulesDisabled(async (ctx) => {
      const db = ctx.firestore();
      for (const path of documentPaths) {
        expect((await getDoc(doc(db, path))).exists(), path).toBe(true);
      }
    });
  });
});

describe.each(clients)("%s", (_name, getDb) => {
  it.each(documentPaths)("cannot read %s", async (path) => {
    await assertFails(getDoc(doc(getDb(), path)));
  });

  it.each(documentPaths)("cannot create or overwrite %s", async (path) => {
    await assertFails(setDoc(doc(getDb(), path), { hacked: true }));
  });

  it.each(documentPaths)("cannot update %s", async (path) => {
    await assertFails(updateDoc(doc(getDb(), path), { hacked: true }));
  });

  it.each(documentPaths)("cannot delete %s", async (path) => {
    await assertFails(deleteDoc(doc(getDb(), path)));
  });

  it.each([...new Set(documentPaths.map(collectionPath))])("cannot list %s", async (path) => {
    await assertFails(getDocs(collection(getDb(), path)));
  });

  it("cannot run a collection group query on members", async () => {
    await assertFails(
      getDocs(query(collectionGroup(getDb(), "members"), where("uid", "==", "u1"))),
    );
  });

  it("cannot list the trips it belongs to", async () => {
    await assertFails(
      getDocs(query(collection(getDb(), "trips"), where("memberUids", "array-contains", "u1"))),
    );
  });
});

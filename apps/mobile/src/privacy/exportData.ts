import type { ApiClient } from "@voyagami/api-client";

/** Downloads the person's data as a ZIP and opens the share sheet. Returns false when this build cannot save files. */
export async function exportMyData(api: ApiClient): Promise<boolean> {
  const { data } = await api.GET("/v1/me/export", { parseAs: "arrayBuffer" });
  if (!data) throw new Error("export failed");
  let FileSystem: typeof import("expo-file-system");
  let Sharing: typeof import("expo-sharing");
  try {
    // Loaded only now so a build made before these modules were added shows a message instead of crashing.
    /* eslint-disable @typescript-eslint/no-require-imports */
    FileSystem = require("expo-file-system") as typeof FileSystem;
    Sharing = require("expo-sharing") as typeof Sharing;
    /* eslint-enable @typescript-eslint/no-require-imports */
  } catch {
    return false;
  }
  const file = new FileSystem.File(FileSystem.Paths.cache, "voyagami-data.zip");
  if (file.exists) file.delete();
  file.create();
  file.write(new Uint8Array(data as ArrayBuffer));
  await Sharing.shareAsync(file.uri, { mimeType: "application/zip", UTI: "public.zip-archive" });
  return true;
}

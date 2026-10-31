import type { ApiClient } from "@voyagami/api-client";

const STOP_CATEGORY = {
  see: "sights",
  eat: "food",
  stay: "stay",
  do: "activity",
  shop: "shopping",
  nightlife: "other",
} as const;

export interface CatalogPlaceLike {
  name: string;
  lat: number;
  lng: number;
  address?: string;
  group: keyof typeof STOP_CATEGORY;
}

/** Adds a catalogue place as a stop on `date`, or the trip's first day. Returns false when the trip has no dates. */
export async function addCatalogPlace(
  api: ApiClient,
  tripId: string,
  place: CatalogPlaceLike,
  date?: string,
): Promise<boolean> {
  let day = date;
  if (!day) {
    const trip = await api.unwrap(api.GET("/v1/trips/{tripId}", { params: { path: { tripId } } }));
    day = trip.dates?.start;
  }
  if (!day) return false;
  await api.unwrap(
    api.POST("/v1/trips/{tripId}/stops", {
      params: { path: { tripId } },
      body: {
        date: day,
        place: {
          name: place.name,
          lat: place.lat,
          lng: place.lng,
          ...(place.address ? { address: place.address } : null),
          category: STOP_CATEGORY[place.group],
        },
      },
    }),
  );
  return true;
}

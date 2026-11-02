// The app's analytics events and their properties: the VOY-224 sheet
// (https://docs.google.com/spreadsheets/d/1vyrppzTQZqM03b-aFLito65Yf6gXGaU1cXuUFrU7iQM).
// Keep in step with analytics.Names in services/api. Property values are short
// identifiers, numbers or booleans only: never names, places, amounts or text.
// Server-only events (billing, limit_reached, ask_draft_created) are not here.

type Id = string;
type Phase = "before" | "during" | "after";
type RecList = "picks" | "stays" | "experiences" | "explore";

export interface EventMap {
  app_installed: undefined;
  app_open: { coldStart: boolean };
  screen_viewed: { route: string; previousRoute?: string };
  sign_in_started: { method: "phone_otp" };
  sign_in: { isNewUser: boolean };
  sign_out: undefined;
  onboarding_step_completed: { step: "region" | "style" | "connect"; skipped: boolean };
  onboarding_completed: { stepsSkipped: number };
  trip_created: {
    tripId: Id;
    destinations: number;
    solo?: boolean;
    source: "home" | "onboarding" | "import" | "ask";
    international?: boolean;
    daysUntilStart?: number;
  };
  trip_opened: { tripId: Id; role: "owner" | "editor" | "viewer"; phase?: Phase };
  trip_deleted: { tripId: Id };
  offline_plan_saved: { tripId: Id };
  sync_failed: { reason: "conflict" | "network" | "forbidden" | "server"; queued?: number };
  place_added: {
    tripId: Id;
    source: "search" | "picks" | "map" | "ask" | "import" | "custom";
    placeGroup?: string;
    dayIndex?: number;
  };
  place_removed: { tripId: Id };
  best_order_applied: { tripId: Id; stops: number };
  itinerary_reordered: { tripId: Id };
  map_opened: { tripId: Id; offline: boolean };
  booking_added: { tripId: Id; kind: string };
  booking_imported: {
    tripId: Id;
    success: boolean;
    inputType: "paste" | "pdf" | "image" | "email";
    kind?: string;
  };
  invite_sent: {
    tripId: Id;
    channel: "share_sheet" | "copy_link" | "qr";
    role?: "editor" | "viewer";
  };
  crew_joined: { tripId: Id; via: "link" | "code" | "qr"; crewSize?: number };
  crew_role_changed: { tripId: Id; role: string };
  poll_created: { tripId: Id; options: number };
  poll_voted: { tripId: Id };
  comment_added: { tripId: Id; target: "trip" | "stop" };
  live_location_started: { tripId: Id; mode: "live" | "meet_point"; durationMin: number };
  expense_added: {
    tripId: Id;
    split: string;
    participants: number;
    foreignCurrency: boolean;
    fromBooking?: boolean;
  };
  settle_up_recorded: { tripId: Id; method: "cash" | "bank_transfer" | "payment_app" | "other" };
  journal_entry_added: { tripId: Id; photos: number; phase?: Phase };
  photo_added: { tripId: Id; count: number; exifKept: boolean };
  recap_viewed: { tripId: Id; completed: boolean };
  recap_shared: { tripId: Id; format: "link" | "video" | "image" };
  place_searched: { surface: "add_place" | "explore"; results: number };
  recs_shown: { list: RecList; count: number; hasSponsored: boolean; tripId?: Id };
  rec_tapped: { list: RecList; position: number; sponsored: boolean; placeId?: Id };
  sponsored_hidden: { list: RecList };
  ask_draft_accepted: { tripId: Id; kept: number };
  paywall_shown: { moment: string };
  paywall_dismissed: { moment: string };
  paywall_converted: { moment: string; product?: string };
  push_opened: { kind: string };
  notification_setting_changed: { kind: string; enabled: boolean };
  data_export_requested: undefined;
  help_opened: undefined;
}

export type EventName = keyof EventMap;

/** The property argument: none for events without properties, required otherwise. */
export type EventArgs<K extends EventName> = EventMap[K] extends undefined
  ? []
  : [props: EventMap[K]];

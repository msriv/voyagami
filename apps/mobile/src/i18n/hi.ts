import type { en } from "./en";

/** Recursively optional, so a language can ship a few screens at a time. */
export type Partial_<T> = { [K in keyof T]?: T[K] extends string ? string : Partial_<T[K]> };

/**
 * Hindi, the proof language (VOY-206). Anything left out falls back to English. The namespaces in
 * REQUIRED_NAMESPACES below must be complete; the i18n test fails if one has a missing key.
 */
export const hi: Partial_<typeof en> = {
  tabs: {
    home: "होम",
    explore: "खोजें",
    profile: "प्रोफ़ाइल",
  },
  home: {
    quickAdd: "{{trip}} में जल्दी जोड़ें",
    welcomeNamed: "स्वागत है, {{name}}",
    welcome: "स्वागत है",
    goodMorning: "सुप्रभात",
    goodMorningNamed: "सुप्रभात, {{name}}",
    goodAfternoon: "नमस्ते",
    goodAfternoonNamed: "नमस्ते, {{name}}",
    goodEvening: "शुभ संध्या",
    goodEveningNamed: "शुभ संध्या, {{name}}",
    profileButton: "अपनी प्रोफ़ाइल खोलें",
    quick: {
      place: "{{trip}} में जगह जोड़ें",
      booking: "{{trip}} में बुकिंग जोड़ें",
      todo: "{{trip}} की टू-डू सूची खोलें",
      expense: "{{trip}} में ख़र्च जोड़ें",
      placeShort: "जगह",
      bookingShort: "बुकिंग",
      todoShort: "टू-डू",
      expenseShort: "ख़र्च",
    },
    brand: "voyagami",
    emptyTitle: "आपका पासपोर्ट अभी ख़ाली है।",
    emptyPlanTitle: "सफ़र की योजना बनाएँ",
    emptyPlanBody: "वीकेंड की छुट्टी हो या विदेश का बड़ा सफ़र",
    whereNext: "अगला सफ़र कहाँ का?",
    upNext: "अगला",
    daysToGo_one: "दिन बाक़ी",
    daysToGo_other: "दिन बाक़ी",
    upcoming: "आने वाले",
    past: "पुराने सफ़र",
    newTrip: "नया सफ़र",
    joinTrip: "किसी सफ़र से जुड़ें",
    inDays_one: "{{count}} दिन में",
    inDays_other: "{{count}} दिनों में",
    startsToday: "आज शुरू",
    ongoing: "अभी चल रहा है",
    dateRange: "{{start}} से {{end}}",
    flexibleDays_one: "{{count}} दिन, तारीख़ें तय नहीं",
    flexibleDays_other: "{{count}} दिन, तारीख़ें तय नहीं",
    listSeparator: ", ",
    loading: "आपके सफ़र लोड हो रहे हैं",
    loadError: "हम आपके सफ़र लोड नहीं कर पाए।",
    retry: "फिर कोशिश करें",
  },
  profile: {
    notifications: "सूचनाएँ",
    privacy: "डेटा और गोपनीयता",
    help: "मदद",
    plan: "प्लान",
    homeRegion: "आपका देश",
    signOut: "साइन आउट",
    resetApp: "ऐप रीसेट करें और फिर से शुरू करें",
    traveller: "यात्री",
    sectionHome: "घर और क्षेत्र",
    sectionData: "आपका डेटा",
    units: "इकाइयाँ",
    unit_km: "किलोमीटर",
    unit_mi: "मील",
    unit_c: "सेल्सियस",
    unit_f: "फ़ारेनहाइट",
    unit_12h: "12 घंटे की घड़ी",
    unit_24h: "24 घंटे की घड़ी",
    dataRegion: "आपका डेटा मुंबई, भारत में रखा गया है",
    dataNote: "सफ़र, ख़र्च और जर्नल की प्रविष्टियाँ इसी क्षेत्र में रहती हैं।",
    appVersion: "Voyagami",
  },
};

/** Namespaces that must have every key. The rest may be partly translated. */
export const REQUIRED_NAMESPACES = ["tabs", "home", "profile"] as const;

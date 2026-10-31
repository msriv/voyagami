import { useLocalSearchParams, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { parsePin } from "@/trips/stops";
import { Button, Chip, IconButton, Input, Text } from "@/ui";
import { Icon } from "@/ui/icons";

const categories = [
  "food",
  "sights",
  "activity",
  "shopping",
  "stay",
  "transport",
  "other",
] as const;
type Category = (typeof categories)[number];
type SearchResult = {
  name: string;
  address?: string;
  lat: number;
  lng: number;
  category?: Category;
};

/** Add a place by hand. Search and pasted links join this screen as more tabs later. */
export default function AddPlaceScreen() {
  const { t } = useTranslation();
  const { colors, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id, date } = useLocalSearchParams<{ id: string; date: string }>();
  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [category, setCategory] = useState<Category | undefined>();
  const [lat, setLat] = useState("");
  const [lng, setLng] = useState("");
  const [note, setNote] = useState("");
  const [query, setQuery] = useState("");
  const [found, setFound] = useState<SearchResult[]>([]);
  const [link, setLink] = useState("");
  const [linkNote, setLinkNote] = useState<string | undefined>();
  const [searchNote, setSearchNote] = useState<string | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const q = query.trim();
    if (q.length < 2) {
      setFound([]);
      setSearchNote(undefined);
      return;
    }
    let live = true;
    const timer = setTimeout(() => {
      api
        .unwrap(api.GET("/v1/places/search", { params: { query: { q } } }))
        .then((r) => {
          if (!live) return;
          setFound(r.results);
          setSearchNote(r.results.length ? undefined : t("addPlace.search.none"));
        })
        .catch(() => {
          if (!live) return;
          setFound([]);
          setSearchNote(t("addPlace.search.failed"));
        });
    }, 350);
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [query, api, t]);

  function pick(r: SearchResult) {
    setName(r.name);
    setAddress(r.address ?? "");
    setCategory(r.category);
    setLat(String(r.lat));
    setLng(String(r.lng));
    setQuery("");
    setFound([]);
  }

  async function readLink() {
    setLinkNote(undefined);
    try {
      const r = await api.unwrap(api.POST("/v1/places/resolve", { body: { url: link.trim() } }));
      pick(r);
      setLink("");
    } catch (e) {
      setLinkNote(t(e instanceof NetworkError ? "addPlace.errors.offline" : "addPlace.link.none"));
    }
  }

  async function save() {
    if (!name.trim()) return setError(t("addPlace.errors.name"));
    const pin = parsePin(lat, lng);
    if (!pin) return setError(t("addPlace.errors.pin"));
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/stops", {
          params: { path: { tripId: id } },
          body: {
            date,
            place: {
              name: name.trim(),
              ...pin,
              ...(address.trim() ? { address: address.trim() } : null),
              ...(category ? { category } : null),
            },
            ...(note.trim() ? { note: note.trim() } : null),
          },
        }),
      );
      router.back();
    } catch (e) {
      setError(
        t(e instanceof NetworkError ? "addPlace.errors.offline" : "addPlace.errors.generic"),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView
        contentContainerStyle={{ padding: 20, gap: 16 }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("addPlace.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("addPlace.title")}
          </Text>
        </View>
        <Input
          label={t("addPlace.search.label")}
          value={query}
          onChangeText={setQuery}
          maxLength={100}
          autoCorrect={false}
        />
        {searchNote ? <Text color="muted">{searchNote}</Text> : null}
        {found.map((r, i) => (
          <Pressable
            key={`${r.name}-${i}`}
            accessibilityRole="button"
            accessibilityLabel={r.address ? `${r.name}, ${r.address}` : r.name}
            onPress={() => pick(r)}
            style={{ minHeight: hitArea.min, justifyContent: "center", gap: 2 }}
          >
            <Text variant="bodyStrong">{r.name}</Text>
            {r.address ? (
              <Text variant="caption" color="muted" numberOfLines={2}>
                {r.address}
              </Text>
            ) : null}
          </Pressable>
        ))}
        <Input
          label={t("addPlace.link.label")}
          value={link}
          onChangeText={setLink}
          maxLength={2000}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="url"
        />
        {link.trim() ? (
          <Button
            label={t("addPlace.link.find")}
            variant="secondary"
            onPress={() => void readLink()}
          />
        ) : null}
        {linkNote ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {linkNote}
          </Text>
        ) : null}
        <Text color="muted">{t("addPlace.intro")}</Text>

        <Input
          label={t("addPlace.nameLabel")}
          value={name}
          onChangeText={setName}
          maxLength={200}
        />
        <Input
          label={t("addPlace.addressLabel")}
          value={address}
          onChangeText={setAddress}
          maxLength={300}
        />

        <View style={{ gap: 6 }}>
          <Text variant="bodyStrong">{t("addPlace.categoryLabel")}</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {categories.map((c) => (
              <Chip
                key={c}
                label={t(`addPlace.categories.${c}`)}
                selected={c === category}
                onPress={() => setCategory(c === category ? undefined : c)}
              />
            ))}
          </View>
        </View>

        <View style={{ gap: 8 }}>
          <Text variant="bodyStrong">{t("addPlace.pinHeading")}</Text>
          <Text variant="caption">{t("addPlace.pinHint")}</Text>
          <Input
            label={t("addPlace.latLabel")}
            value={lat}
            onChangeText={setLat}
            keyboardType="numbers-and-punctuation"
          />
          <Input
            label={t("addPlace.lngLabel")}
            value={lng}
            onChangeText={setLng}
            keyboardType="numbers-and-punctuation"
          />
        </View>

        <Input
          label={t("addPlace.noteLabel")}
          value={note}
          onChangeText={setNote}
          maxLength={2000}
          multiline
        />
        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        <Button label={t("addPlace.save")} loading={busy} onPress={() => void save()} />
      </ScrollView>
    </SafeAreaView>
  );
}

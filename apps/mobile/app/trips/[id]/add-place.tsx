import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { parsePin } from "@/trips/stops";
import { Button, Chip, IconButton, Input, Text } from "@/ui";

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

/** Add a place by hand. Search and pasted links join this screen as more tabs later. */
export default function AddPlaceScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id, date } = useLocalSearchParams<{ id: string; date: string }>();
  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [category, setCategory] = useState<Category | undefined>();
  const [lat, setLat] = useState("");
  const [lng, setLng] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

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
            <Text variant="bodyStrong">{"‹"}</Text>
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("addPlace.title")}
          </Text>
        </View>
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

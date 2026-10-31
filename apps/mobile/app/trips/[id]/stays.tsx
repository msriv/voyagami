import { useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useEffect, useState } from "react";
import { Linking, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, IconButton, Text } from "@/ui";
import { Icon } from "@/ui/icons";

interface Link {
  partnerId: string;
  name: string;
  url: string;
}

type State =
  | { kind: "loading" }
  | { kind: "ready"; links: Link[] }
  | { kind: "noDestination" }
  | { kind: "error" };

/**
 * Where to stay: one button per booking partner. Tapping leaves the app for the partner's site;
 * nothing is booked or paid here. After booking, add the stay to Bookings.
 */
export default function StaysScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });

  const load = useCallback(async () => {
    try {
      const { links } = await api.unwrap(
        api.GET("/v1/trips/{tripId}/stays/links", { params: { path: { tripId: id } } }),
      );
      setState({ kind: "ready", links });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "bad_request"
          ? { kind: "noDestination" }
          : { kind: "error" },
      );
    }
  }, [api, id]);

  useEffect(() => {
    void load();
  }, [load]);

  const open = (link: Link) => {
    // Counting the tap is best effort: it must never stop the person leaving for the partner.
    void api
      .POST("/v1/trips/{tripId}/stays/clicks", {
        params: { path: { tripId: id } },
        body: { partnerId: link.partnerId },
      })
      .catch(() => undefined);
    void Linking.openURL(link.url);
  };

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          <IconButton
            accessibilityLabel={t("stays.back")}
            variant="secondary"
            onPress={() => router.back()}
          >
            <Icon name="back" color={colors.ink} />
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("stays.title")}
          </Text>
        </View>

        <Text>{t("stays.note")}</Text>

        {state.kind === "loading" ? <Text color="muted">{t("stays.loading")}</Text> : null}
        {state.kind === "noDestination" ? (
          <Text accessibilityLiveRegion="polite">{t("stays.noDestination")}</Text>
        ) : null}
        {state.kind === "error" ? (
          <Card>
            <Text accessibilityLiveRegion="polite">{t("stays.loadError")}</Text>
            <Button label={t("stays.retry")} variant="secondary" onPress={() => void load()} />
          </Card>
        ) : null}
        {state.kind === "ready" && state.links.length === 0 ? (
          <Text accessibilityLiveRegion="polite">{t("stays.none")}</Text>
        ) : null}
        {state.kind === "ready"
          ? state.links.map((link) => (
              <Button
                key={link.partnerId}
                label={t("stays.openPartner", { name: link.name })}
                onPress={() => open(link)}
              />
            ))
          : null}

        <Card style={{ gap: 8 }}>
          <Text variant="bodyStrong" heading>
            {t("stays.afterTitle")}
          </Text>
          <Text>{t("stays.afterBody")}</Text>
          <Button
            label={t("stays.addBooking")}
            variant="secondary"
            onPress={() => router.push(`/trips/${id}/add-booking`)}
          />
        </Card>
      </ScrollView>
    </SafeAreaView>
  );
}

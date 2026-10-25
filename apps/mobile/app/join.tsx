import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, IconButton, Input, Text } from "@/ui";

/**
 * Join a trip with the code from an invite link. The link opens this screen with the code filled in.
 * shortcut: someone who is signed out when they open the link types the code in again after
 * signing in, until pending links are kept across sign-in.
 */
export default function JoinScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const params = useLocalSearchParams<{ token?: string }>();
  const [token, setToken] = useState(params.token ?? "");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  async function join() {
    if (!token.trim()) return setError(t("join.errors.empty"));
    setBusy(true);
    setError(undefined);
    try {
      const trip = await api.unwrap(
        api.POST("/v1/invites/join", { body: { token: token.trim() } }),
      );
      router.replace(`/trips/${trip.id}`);
    } catch (e) {
      setError(
        t(
          e instanceof ApiError && e.code === "not_found"
            ? "join.errors.invalid"
            : e instanceof NetworkError
              ? "join.errors.offline"
              : "join.errors.generic",
        ),
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
            accessibilityLabel={t("join.back")}
            variant="secondary"
            onPress={() => (router.canGoBack() ? router.back() : router.replace("/"))}
          >
            <Text variant="bodyStrong">{"‹"}</Text>
          </IconButton>
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("join.title")}
          </Text>
        </View>
        <Text color="muted">{t("join.intro")}</Text>
        <Input
          label={t("join.codeLabel")}
          value={token}
          onChangeText={setToken}
          autoCapitalize="none"
          autoCorrect={false}
          maxLength={200}
        />
        {error ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        <Button label={t("join.button")} loading={busy} onPress={() => void join()} />
      </ScrollView>
    </SafeAreaView>
  );
}

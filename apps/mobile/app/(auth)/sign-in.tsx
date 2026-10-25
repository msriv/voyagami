import { callingCodeFor, toE164 } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useRouter } from "expo-router";
import { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { signInErrorKey, useAuth, withDetail } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, IconButton, Input, Text } from "@/ui";

export default function SignInScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api, setPendingPhone } = useAuth();
  const region = getLocales()[0]?.regionCode ?? null;
  const code = callingCodeFor(region);
  const [phone, setPhone] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  async function sendCode() {
    const e164 = toE164(phone, region);
    if (!e164) {
      setError(t("signIn.errors.invalidNumber"));
      return;
    }
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(api.POST("/v1/auth/otp/request", { body: { phone: e164 } }));
      setPendingPhone(e164);
      router.push("/code");
    } catch (e) {
      setError(withDetail(t(`signIn.errors.${signInErrorKey(e)}`), e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView
          keyboardShouldPersistTaps="handled"
          contentContainerStyle={{ padding: 24, gap: 22, flexGrow: 1 }}
        >
          <View style={{ flexDirection: "row" }}>
            <IconButton
              accessibilityLabel={t("signIn.back")}
              variant="secondary"
              onPress={() => router.back()}
            >
              <Text variant="bodyStrong">{"‹"}</Text>
            </IconButton>
          </View>
          <View style={{ gap: 8 }}>
            <Text variant="display" heading style={{ fontSize: 36, lineHeight: 38 }}>
              {t("signIn.title")}
            </Text>
            <Text color="muted">{t("signIn.intro")}</Text>
          </View>
          <Input
            label={t("signIn.phoneLabel")}
            hint={code ? t("signIn.phoneHint", { code }) : t("signIn.phoneHintNoRegion")}
            error={error}
            value={phone}
            onChangeText={setPhone}
            keyboardType="phone-pad"
            textContentType="telephoneNumber"
            autoComplete="tel"
            returnKeyType="send"
            onSubmitEditing={sendCode}
          />
          <Button label={t("signIn.sendCode")} onPress={sendCode} loading={busy} fullWidth />
          <View style={{ flex: 1 }} />
          <Text variant="caption" style={{ textAlign: "center" }}>
            {t("signIn.consent")}
          </Text>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

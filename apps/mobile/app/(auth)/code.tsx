import { getLocales } from "expo-localization";
import { Redirect, useRouter } from "expo-router";
import { useEffect, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { signInErrorKey, useAuth, withDetail } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Input, Text } from "@/ui";

const resendSeconds = 30;

export default function CodeScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const router = useRouter();
  const { api, signIn, pendingPhone } = useAuth();
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [wait, setWait] = useState(resendSeconds);

  useEffect(() => {
    if (wait <= 0) return;
    const id = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(id);
  }, [wait]);

  if (!pendingPhone) return <Redirect href="/sign-in" />;

  async function verify() {
    if (!pendingPhone) return;
    setBusy(true);
    setError(undefined);
    const locale = getLocales()[0];
    try {
      const session = await api.unwrap(
        api.POST("/v1/auth/otp/verify", {
          body: {
            phone: pendingPhone,
            code,
            ...(locale?.regionCode ? { region: locale.regionCode } : null),
            ...(locale?.languageTag ? { language: locale.languageTag } : null),
          },
        }),
      );
      await signIn({ accessToken: session.accessToken, refreshToken: session.refreshToken });
    } catch (e) {
      setError(withDetail(t(`signIn.errors.${signInErrorKey(e)}`), e));
    } finally {
      setBusy(false);
    }
  }

  async function resend() {
    if (!pendingPhone) return;
    setError(undefined);
    try {
      await api.unwrap(api.POST("/v1/auth/otp/request", { body: { phone: pendingPhone } }));
      setWait(resendSeconds);
    } catch (e) {
      setError(withDetail(t(`signIn.errors.${signInErrorKey(e)}`), e));
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
          contentContainerStyle={{ padding: 24, gap: 20, flexGrow: 1, justifyContent: "center" }}
        >
          <View style={{ gap: 8 }}>
            <Text variant="display" heading>
              {t("signIn.codeTitle")}
            </Text>
            <Text color="muted">{t("signIn.codeIntro", { phone: pendingPhone })}</Text>
          </View>
          <Input
            label={t("signIn.codeLabel")}
            error={error}
            value={code}
            onChangeText={(text) => setCode(text.replace(/\D/g, "").slice(0, 6))}
            keyboardType="number-pad"
            textContentType="oneTimeCode"
            autoComplete="sms-otp"
            maxLength={6}
            autoFocus
            returnKeyType="done"
            onSubmitEditing={verify}
          />
          <Button
            label={t("signIn.verify")}
            onPress={verify}
            loading={busy}
            disabled={code.length !== 6}
            fullWidth
          />
          <Button
            label={wait > 0 ? t("signIn.resendIn", { seconds: wait }) : t("signIn.resend")}
            accessibilityLabel={t("signIn.resend")}
            variant="ghost"
            onPress={resend}
            disabled={wait > 0}
            fullWidth
          />
          <Button
            label={t("signIn.changeNumber")}
            variant="ghost"
            onPress={() => router.back()}
            fullWidth
          />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

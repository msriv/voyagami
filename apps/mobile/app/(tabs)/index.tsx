import { useEffect, useState } from "react";
import { View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Text } from "@/ui";

export default function HomeScreen() {
  const { t } = useTranslation();
  const { api } = useAuth();
  const [name, setName] = useState<string | undefined>();

  useEffect(() => {
    let alive = true;
    api
      .unwrap(api.GET("/v1/me"))
      .then((me) => alive && setName(me.displayName))
      // Home still works offline; the greeting just has no name.
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, [api]);

  return (
    <View style={{ flex: 1, padding: 24, gap: 8 }} accessibilityLabel={t("tabs.home")}>
      <Text variant="display" heading>
        {name ? t("home.welcomeNamed", { name }) : t("home.welcome")}
      </Text>
      <Text color="muted">{t("home.signedIn")}</Text>
    </View>
  );
}

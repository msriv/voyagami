import { View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Button } from "@/ui";

export default function ProfileScreen() {
  const { t } = useTranslation();
  const { signOut } = useAuth();
  return (
    <View style={{ flex: 1, padding: 24 }} accessibilityLabel={t("tabs.profile")}>
      <Button label={t("profile.signOut")} variant="secondary" onPress={() => void signOut()} />
    </View>
  );
}

import { View } from "react-native";

import { useTranslation } from "@/i18n";

export default function HomeScreen() {
  const { t } = useTranslation();
  return <View style={{ flex: 1 }} accessibilityLabel={t("tabs.home")} />;
}

import { Redirect } from "expo-router";
import { useState } from "react";
import { ScrollView, View } from "react-native";

import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import {
  BrushCircle,
  Button,
  Card,
  Chip,
  IconButton,
  Input,
  Postmark,
  Seal,
  Sheet,
  Text,
  Toast,
  WashiTape,
  WavePattern,
  type TextVariant,
} from "@/ui";

const textVariants: TextVariant[] = [
  "display",
  "title",
  "body",
  "bodyStrong",
  "caption",
  "accent",
  "handwriting",
];

/** Dev-only: shows every UI kit component and state. In release builds it sends you home. */
export default function GalleryScreen() {
  if (!__DEV__) return <Redirect href="/" />;
  return <Gallery />;
}

function Gallery() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const [chip, setChip] = useState(true);
  const [name, setName] = useState("");
  const [sheet, setSheet] = useState(false);
  const [toast, setToast] = useState(false);
  const x = <Text color="ink">{"\u00d7"}</Text>;

  return (
    <ScrollView
      style={{ backgroundColor: colors.washi }}
      contentContainerStyle={{ padding: 16, gap: 24, paddingBottom: 48 }}
    >
      <Text variant="display" heading>
        {t("gallery.title")}
      </Text>

      <Section title={t("gallery.textStyles")}>
        {textVariants.map((v) => (
          <Text key={v} variant={v}>
            {v}
          </Text>
        ))}
      </Section>

      <Section title={t("gallery.buttons")}>
        <Button label={t("gallery.primary")} />
        <Button label={t("gallery.accent")} variant="accent" />
        <Button label={t("gallery.secondary")} variant="secondary" />
        <Button label={t("gallery.ghost")} variant="ghost" />
        <Button label={t("gallery.disabled")} disabled />
        <Button label={t("gallery.loading")} loading />
        <View style={{ flexDirection: "row" }}>
          <IconButton accessibilityLabel={t("gallery.iconButton")}>{x}</IconButton>
        </View>
      </Section>

      <Section title={t("gallery.cards")}>
        <Card>
          <Text>{t("gallery.cardBody")}</Text>
        </Card>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
          <Chip label={t("gallery.chipOn")} selected={chip} onPress={() => setChip(!chip)} />
          <Chip label={t("gallery.chipOff")} selected={false} />
          <Chip label={t("gallery.chipDisabled")} selected={false} disabled />
        </View>
      </Section>

      <Section title={t("gallery.inputs")}>
        <Input
          label={t("gallery.inputLabel")}
          hint={t("gallery.inputHint")}
          value={name}
          onChangeText={setName}
        />
        <Input label={t("gallery.inputLabel")} error={t("gallery.inputError")} value="" />
        <Input label={t("gallery.inputDisabled")} editable={false} value="" />
      </Section>

      <Section title={t("gallery.overlays")}>
        <Button label={t("gallery.openSheet")} variant="secondary" onPress={() => setSheet(true)} />
        <Button label={t("gallery.showToast")} variant="secondary" onPress={() => setToast(true)} />
        {toast ? (
          <Toast
            message={t("gallery.toastMessage")}
            dismissLabel={t("gallery.dismiss")}
            dismissIcon={<Text color="white">{"\u00d7"}</Text>}
            onDismiss={() => setToast(false)}
          />
        ) : null}
      </Section>

      <Section title={t("gallery.motifs")}>
        <View
          style={{
            height: 72,
            borderRadius: 24,
            overflow: "hidden",
            backgroundColor: colors.indigo,
          }}
        >
          <WavePattern opacity={0.3} />
        </View>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 16, flexWrap: "wrap" }}>
          <Seal accessibilityLabel={t("gallery.sealLabel")} />
          <BrushCircle />
          <WashiTape />
        </View>
        <Postmark city={t("gallery.postmarkCity")} date={t("gallery.postmarkDate")} />
      </Section>

      <Sheet
        visible={sheet}
        onClose={() => setSheet(false)}
        title={t("gallery.sheetTitle")}
        closeLabel={t("gallery.close")}
        closeIcon={x}
      >
        <Text>{t("gallery.sheetBody")}</Text>
      </Sheet>
    </ScrollView>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={{ gap: 12 }}>
      <Text variant="title" heading>
        {title}
      </Text>
      {children}
    </View>
  );
}

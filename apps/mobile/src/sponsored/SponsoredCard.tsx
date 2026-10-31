import { useEffect, useState } from "react";
import { Image, Linking, Pressable, View } from "react-native";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, Card, Text } from "@/ui";
import type { components } from "@voyagami/api-client";

type Card = components["schemas"]["SponsoredCard"];
type Group = components["schemas"]["PlaceGroup"];

/**
 * The one sponsored card for a list, if the server has one. Always labelled "Sponsored", shown on every plan,
 * and hideable for 30 days. If anything fails the card is simply not drawn.
 */
export function SponsoredCard({ tripId, group }: { tripId: string; group?: Group }) {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const [card, setCard] = useState<Card | undefined>();

  useEffect(() => {
    let alive = true;
    setCard(undefined);
    void (async () => {
      try {
        const r = await api.unwrap(
          api.GET("/v1/trips/{tripId}/sponsored", {
            params: { path: { tripId }, query: group ? { group } : {} },
          }),
        );
        if (!alive || !r.card) return;
        setCard(r.card);
        // Counting is best effort and the server counts each person once a day.
        void api
          .POST("/v1/sponsored/{cardId}/impressions", { params: { path: { cardId: r.card.id } } })
          .catch(() => undefined);
      } catch {
        /* no card */
      }
    })();
    return () => {
      alive = false;
    };
  }, [api, tripId, group]);

  if (!card) return null;

  const open = () => {
    void api
      .POST("/v1/sponsored/{cardId}/clicks", { params: { path: { cardId: card.id } } })
      .catch(() => undefined);
    void Linking.openURL(card.url);
  };
  const hide = () => {
    setCard(undefined);
    void api
      .POST("/v1/sponsored/{cardId}/hide", { params: { path: { cardId: card.id } } })
      .catch(() => undefined);
  };

  return (
    <Card tone="tinted" style={{ gap: 8 }}>
      <View style={{ flexDirection: "row", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
        <Text variant="bodyStrong">{t("sponsored.label")}</Text>
        <Text variant="caption">{card.sponsor}</Text>
      </View>
      <Pressable
        accessibilityRole="link"
        accessibilityLabel={t("sponsored.open", { title: card.title })}
        hitSlop={8}
        onPress={open}
        style={{ gap: 8 }}
      >
        {card.imageUrl ? (
          <Image
            source={{ uri: card.imageUrl }}
            accessibilityIgnoresInvertColors
            style={{ height: 140, borderRadius: 16, backgroundColor: colors.line }}
          />
        ) : null}
        <Text variant="bodyStrong" heading>
          {card.title}
        </Text>
        {card.body ? <Text>{card.body}</Text> : null}
      </Pressable>
      <Button label={t("sponsored.hide")} variant="ghost" onPress={hide} />
    </Card>
  );
}

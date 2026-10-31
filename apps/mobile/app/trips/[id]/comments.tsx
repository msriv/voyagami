import { formatDate, formatTime } from "@voyagami/locale";
import { getLocales } from "expo-localization";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { useTripChanges } from "@/trips/useTripChanges";
import type { Trip } from "@/trips/summary";
import { Button, Card, IconButton, Input, Sheet, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Comment = components["schemas"]["Comment"];

type State =
  | { kind: "loading" }
  | {
      kind: "ready";
      trip: Trip;
      comments: Comment[];
      names: Record<string, string>;
      me: string;
    }
  | { kind: "notFound" }
  | { kind: "error" };

/** Comments on one stop (`stopId`) or on the whole trip when there is none. */
export default function CommentsScreen() {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id, stopId, title } = useLocalSearchParams<{
    id: string;
    stopId?: string;
    title?: string;
  }>();
  const locale = getLocales()[0]?.languageTag ?? "en";
  const targetType = stopId ? "stop" : "trip";
  const targetId = stopId ?? id;
  const [state, setState] = useState<State>({ kind: "loading" });
  const [draft, setDraft] = useState("");
  const [open, setOpen] = useState<Comment | undefined>();
  const [edit, setEdit] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const path = { params: { path: { tripId: id } } };
      const [trip, { comments }, { members }, me] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", path)),
        api.unwrap(
          api.GET("/v1/trips/{tripId}/comments", {
            params: { path: { tripId: id }, query: { targetType, targetId } },
          }),
        ),
        api.unwrap(api.GET("/v1/trips/{tripId}/members", path)),
        api.unwrap(api.GET("/v1/me")),
      ]);
      const names = Object.fromEntries(members.map((m) => [m.uid, m.displayName ?? ""]));
      setState({ kind: "ready", trip, comments, names, me: me.id });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id, targetType, targetId]);

  useTripChanges(id, () => void load());

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("comments.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Icon name="back" color={colors.ink} />
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>
              {state.kind === "notFound" ? t("comments.notFound") : t("comments.loadError")}
            </Text>
            {state.kind === "error" ? (
              <Button label={t("comments.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, comments, names, me } = state;
  const fail = (e: unknown) =>
    setError(t(e instanceof NetworkError ? "comments.errors.offline" : "comments.errors.generic"));

  async function send() {
    const body = draft.trim();
    if (!body) return;
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/comments", {
          params: { path: { tripId: id } },
          body: { targetType, targetId, body },
        }),
      );
      setDraft("");
      await load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function save(c: Comment) {
    const body = edit.trim();
    if (!body) return;
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.PATCH("/v1/trips/{tripId}/comments/{commentId}", {
          params: { path: { tripId: id, commentId: c.id } },
          body: { body },
        }),
      );
      setOpen(undefined);
      await load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function remove(c: Comment) {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/comments/{commentId}", {
          params: { path: { tripId: id, commentId: c.id } },
        }),
      );
      setOpen(undefined);
      await load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  const canDelete = (c: Comment) => c.authorUid === me || trip.role === "owner";

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("comments.title")}
          </Text>
        </View>
        <Text color="muted">{title ?? trip.title}</Text>

        {comments.length === 0 ? <Text color="muted">{t("comments.empty")}</Text> : null}
        {comments.map((c) => {
          const who = c.authorUid === me ? t("comments.you") : names[c.authorUid] || "?";
          const when = new Date(c.createdAt);
          const mine = c.authorUid === me;
          return (
            <Card key={c.id}>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={t("comments.commentLabel", { name: who, body: c.body })}
                disabled={!mine && !canDelete(c)}
                onPress={() => {
                  setError(undefined);
                  setEdit(c.body);
                  setOpen(c);
                }}
                style={{ minHeight: hitArea.min, gap: 2, borderRadius: radius.button }}
              >
                <Text variant="caption">
                  {t("comments.byLine", {
                    name: who,
                    date: formatDate(when, locale, { timeZone: trip.timeZone }),
                    time: formatTime(when, locale, { timeZone: trip.timeZone }),
                  })}
                  {c.editedAt ? ` ${t("comments.edited")}` : ""}
                </Text>
                <Text>{c.body}</Text>
              </Pressable>
            </Card>
          );
        })}

        <Input
          label={t("comments.write")}
          value={draft}
          onChangeText={setDraft}
          multiline
          maxLength={2000}
        />
        {error && !open ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        <Button label={t("comments.send")} loading={busy} onPress={() => void send()} />
      </ScrollView>

      <Sheet
        visible={open !== undefined}
        onClose={() => setOpen(undefined)}
        title={t("comments.manage")}
        closeLabel={t("comments.close")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          {open && open.authorUid === me ? (
            <>
              <Input
                label={t("comments.editLabel")}
                value={edit}
                onChangeText={setEdit}
                multiline
                maxLength={2000}
              />
              <Button label={t("comments.save")} loading={busy} onPress={() => void save(open)} />
            </>
          ) : null}
          {open && canDelete(open) ? (
            <Button
              label={t("comments.delete")}
              variant="secondary"
              loading={busy}
              onPress={() => void remove(open)}
            />
          ) : null}
          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
        </View>
      </Sheet>
    </SafeAreaView>
  );
}

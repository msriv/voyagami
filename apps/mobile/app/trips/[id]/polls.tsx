import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { ScrollView, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { useTripChanges } from "@/trips/useTripChanges";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Input, Sheet, Text } from "@/ui";
import { Icon } from "@/ui/icons";

type Poll = components["schemas"]["Poll"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; polls: Poll[] }
  | { kind: "notFound" }
  | { kind: "error" };

/** Quick crew decisions: anyone votes, owners and editors start and close polls. */
export default function PollsScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [creating, setCreating] = useState(false);
  const [question, setQuestion] = useState("");
  const [labels, setLabels] = useState(["", ""]);
  const [multi, setMulti] = useState(false);
  // The poll that ended in a tie, waiting for someone to pick the winner.
  const [tied, setTied] = useState<string | undefined>();

  const load = useCallback(async () => {
    try {
      const path = { params: { path: { tripId: id } } };
      const [trip, { polls }] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}", path)),
        api.unwrap(api.GET("/v1/trips/{tripId}/polls", path)),
      ]);
      setState({ kind: "ready", trip, polls });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useTripChanges(id, () => void load());

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const back = (
    <IconButton
      accessibilityLabel={t("polls.back")}
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
            <Text>{state.kind === "notFound" ? t("polls.notFound") : t("polls.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("polls.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, polls } = state;
  const canEdit = trip.role !== "viewer";
  const fail = (e: unknown) =>
    setError(t(e instanceof NetworkError ? "polls.errors.offline" : "polls.errors.generic"));

  /** Swaps the answered poll into the list. */
  const replace = (poll: Poll) =>
    setState({ kind: "ready", trip, polls: polls.map((p) => (p.id === poll.id ? poll : p)) });

  async function vote(poll: Poll, optionId: string) {
    const next = !poll.multi
      ? [optionId]
      : poll.myVote.includes(optionId)
        ? poll.myVote.filter((o) => o !== optionId)
        : [...poll.myVote, optionId];
    if (next.length === 0) return;
    setError(undefined);
    try {
      replace(
        await api.unwrap(
          api.POST("/v1/trips/{tripId}/polls/{pollId}/vote", {
            params: { path: { tripId: id, pollId: poll.id } },
            body: { optionIds: next },
          }),
        ),
      );
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) void load();
      else fail(e);
    }
  }

  async function close(poll: Poll, optionId?: string) {
    setBusy(true);
    setError(undefined);
    try {
      const { poll: closed } = await api.unwrap(
        api.POST("/v1/trips/{tripId}/polls/{pollId}/close", {
          params: { path: { tripId: id, pollId: poll.id } },
          body: optionId ? { optionId } : {},
        }),
      );
      setTied(undefined);
      replace(closed);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409 && !optionId) setTied(poll.id);
      else fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function create() {
    const options = labels.map((l) => l.trim()).filter(Boolean);
    if (!question.trim() || options.length < 2) return setError(t("polls.errors.incomplete"));
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.POST("/v1/trips/{tripId}/polls", {
          params: { path: { tripId: id } },
          body: { question: question.trim(), options: options.map((label) => ({ label })), multi },
        }),
      );
      setCreating(false);
      setQuestion("");
      setLabels(["", ""]);
      setMulti(false);
      await load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi }}>
      <ScrollView contentContainerStyle={{ padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
          {back}
          <Text variant="title" heading style={{ flex: 1 }}>
            {t("polls.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>

        {polls.length === 0 ? <Text color="muted">{t("polls.empty")}</Text> : null}
        {polls.map((p) => {
          const closed = p.status === "closed";
          return (
            <Card key={p.id}>
              <Text variant="bodyStrong">{p.question}</Text>
              <Text variant="caption">
                {t(closed ? "polls.closed" : "polls.open", { count: p.voters })}
              </Text>
              {p.options.map((o) => {
                const winner = closed && p.result?.optionId === o.id;
                const text = t("polls.optionVotes", { label: o.label, count: o.votes });
                return (
                  <View key={o.id} style={{ gap: 4 }}>
                    {closed ? (
                      <Text accessible variant={winner ? "bodyStrong" : "body"}>
                        {winner ? t("polls.winner", { text }) : text}
                      </Text>
                    ) : (
                      <Chip
                        label={text}
                        selected={p.myVote.includes(o.id)}
                        onPress={() => (tied === p.id ? void close(p, o.id) : void vote(p, o.id))}
                      />
                    )}
                  </View>
                );
              })}
              {!closed && tied === p.id ? (
                <Text color="vermilion" accessibilityLiveRegion="polite">
                  {t("polls.tie")}
                </Text>
              ) : null}
              {!closed && canEdit && tied !== p.id ? (
                <Button
                  label={t("polls.close")}
                  variant="secondary"
                  loading={busy}
                  onPress={() => void close(p)}
                />
              ) : null}
            </Card>
          );
        })}
        {error && !creating ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
        {canEdit ? (
          <Button
            label={t("polls.new")}
            onPress={() => {
              setError(undefined);
              setCreating(true);
            }}
          />
        ) : null}
      </ScrollView>

      <Sheet
        visible={creating}
        onClose={() => setCreating(false)}
        title={t("polls.new")}
        closeLabel={t("polls.cancel")}
        closeIcon={<Icon name="close" color={colors.ink} />}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          <Input label={t("polls.question")} value={question} onChangeText={setQuestion} />
          {labels.map((l, i) => (
            <Input
              key={i}
              label={t("polls.optionLabel", { n: i + 1 })}
              value={l}
              onChangeText={(v) => setLabels(labels.map((x, j) => (j === i ? v : x)))}
            />
          ))}
          {labels.length < 10 ? (
            <Button
              label={t("polls.addOption")}
              variant="secondary"
              onPress={() => setLabels([...labels, ""])}
            />
          ) : null}
          <Chip label={t("polls.multi")} selected={multi} onPress={() => setMulti(!multi)} />
          {error ? (
            <Text color="vermilion" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
          <Button label={t("polls.create")} loading={busy} onPress={() => void create()} />
        </View>
      </Sheet>
    </SafeAreaView>
  );
}

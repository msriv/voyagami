import * as Linking from "expo-linking";
import { useFocusEffect, useLocalSearchParams, useRouter } from "expo-router";
import { useCallback, useState } from "react";
import { ScrollView, Share, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { ApiError, NetworkError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import type { Trip } from "@/trips/summary";
import { Button, Card, Chip, IconButton, Sheet, Text } from "@/ui";

type Member = components["schemas"]["Member"];
type Invite = components["schemas"]["Invite"];

type State =
  | { kind: "loading" }
  | { kind: "ready"; trip: Trip; members: Member[]; invites: Invite[]; me: string }
  | { kind: "notFound" }
  | { kind: "error" };

/** Who is on the trip, and (for the owner) the invite links that bring more people in. */
export default function CrewScreen() {
  const { t } = useTranslation();
  const { colors } = useTheme();
  const { api } = useAuth();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [open, setOpen] = useState<Member | undefined>();
  const [role, setRole] = useState<"editor" | "viewer">("editor");
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const trip = await api.unwrap(
        api.GET("/v1/trips/{tripId}", { params: { path: { tripId: id } } }),
      );
      const [{ members }, me, invites] = await Promise.all([
        api.unwrap(api.GET("/v1/trips/{tripId}/members", { params: { path: { tripId: id } } })),
        api.unwrap(api.GET("/v1/me")),
        trip.role === "owner"
          ? api
              .unwrap(api.GET("/v1/trips/{tripId}/invites", { params: { path: { tripId: id } } }))
              .then((r) => r.invites)
          : Promise.resolve([] as Invite[]),
      ]);
      setState({ kind: "ready", trip, members, invites, me: me.id });
    } catch (e) {
      setState(
        e instanceof ApiError && e.code === "not_found" ? { kind: "notFound" } : { kind: "error" },
      );
    }
  }, [api, id]);

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  const fail = (e: unknown) =>
    setError(t(e instanceof NetworkError ? "crew.errors.offline" : "crew.errors.generic"));

  const back = (
    <IconButton
      accessibilityLabel={t("crew.back")}
      variant="secondary"
      onPress={() => router.back()}
    >
      <Text variant="bodyStrong">{"‹"}</Text>
    </IconButton>
  );

  if (state.kind !== "ready") {
    return (
      <SafeAreaView style={{ flex: 1, backgroundColor: colors.washi, padding: 20, gap: 16 }}>
        <View style={{ flexDirection: "row" }}>{back}</View>
        {state.kind === "loading" ? null : (
          <View style={{ gap: 12 }}>
            <Text>{state.kind === "notFound" ? t("crew.notFound") : t("crew.loadError")}</Text>
            {state.kind === "error" ? (
              <Button label={t("crew.retry")} variant="secondary" onPress={() => void load()} />
            ) : null}
          </View>
        )}
      </SafeAreaView>
    );
  }

  const { trip, members, invites, me } = state;
  const isOwner = trip.role === "owner";
  const active = members.filter((m) => m.status === "active");
  const liveInvites = invites.filter((i) => !i.revoked);

  async function invite() {
    setBusy(true);
    setError(undefined);
    try {
      const { token } = await api.unwrap(
        api.POST("/v1/trips/{tripId}/invites", {
          params: { path: { tripId: id } },
          body: { role },
        }),
      );
      // The token is shown only now, so it goes straight to the share sheet.
      await Share.share({
        message: t("crew.shareMessage", {
          trip: trip.title,
          link: Linking.createURL("join", { queryParams: { token } }),
          code: token,
        }),
      });
      await load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function revoke(inv: Invite) {
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/invites/{inviteId}", {
          params: { path: { tripId: id, inviteId: inv.id } },
        }),
      );
      setState({ ...state, invites: invites.filter((i) => i.id !== inv.id) } as State);
    } catch (e) {
      fail(e);
    }
  }

  async function setMemberRole(member: Member, next: "editor" | "viewer") {
    setBusy(true);
    setError(undefined);
    try {
      const saved = await api.unwrap(
        api.PATCH("/v1/trips/{tripId}/members/{uid}", {
          params: { path: { tripId: id, uid: member.uid } },
          body: { role: next },
        }),
      );
      setState({
        ...state,
        members: members.map((m) => (m.uid === saved.uid ? saved : m)),
      } as State);
      setOpen(saved);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function removeMember(member: Member) {
    setBusy(true);
    setError(undefined);
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/members/{uid}", {
          params: { path: { tripId: id, uid: member.uid } },
        }),
      );
      setOpen(undefined);
      if (member.uid === me) router.replace("/");
      else await load();
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
            {t("crew.title")}
          </Text>
        </View>
        <Text color="muted">{trip.title}</Text>
        <Button
          label={t("crew.live")}
          variant="secondary"
          onPress={() => router.push(`/trips/${id}/live`)}
        />
        <Button
          label={t("crew.polls")}
          variant="secondary"
          onPress={() => router.push(`/trips/${id}/polls`)}
        />

        {active.map((m) => (
          <Card key={m.uid}>
            <View style={{ flexDirection: "row", alignItems: "center", gap: 12 }}>
              <View style={{ flex: 1 }}>
                <Text variant="bodyStrong">
                  {m.uid === me ? t("crew.you", { name: m.displayName }) : m.displayName}
                </Text>
                <Text variant="caption">{t(`trip.roles.${m.role}`)}</Text>
              </View>
              {(isOwner && m.role !== "owner") || (m.uid === me && m.role !== "owner") ? (
                <Button
                  label={t("crew.manage")}
                  variant="secondary"
                  accessibilityLabel={t("crew.manageName", { name: m.displayName })}
                  onPress={() => {
                    setError(undefined);
                    setOpen(m);
                  }}
                />
              ) : null}
            </View>
          </Card>
        ))}

        {isOwner ? (
          <View style={{ gap: 10 }}>
            <Text variant="title" heading>
              {t("crew.inviteHeading")}
            </Text>
            <View style={{ flexDirection: "row", gap: 8 }}>
              {(["editor", "viewer"] as const).map((r) => (
                <Chip
                  key={r}
                  label={t(`crew.inviteAs.${r}`)}
                  selected={r === role}
                  onPress={() => setRole(r)}
                />
              ))}
            </View>
            <Button label={t("crew.invite")} loading={busy} onPress={() => void invite()} />
            {liveInvites.map((inv) => (
              <Card key={inv.id}>
                <Text>
                  {t("crew.inviteLine", { role: t(`trip.roles.${inv.role}`), count: inv.useCount })}
                </Text>
                <Button
                  label={t("crew.revoke")}
                  variant="secondary"
                  accessibilityLabel={t("crew.revokeFor", { role: t(`trip.roles.${inv.role}`) })}
                  onPress={() => void revoke(inv)}
                />
              </Card>
            ))}
          </View>
        ) : null}

        {error && !open ? (
          <Text color="vermilion" accessibilityLiveRegion="polite">
            {error}
          </Text>
        ) : null}
      </ScrollView>

      <Sheet
        visible={open !== undefined}
        onClose={() => setOpen(undefined)}
        title={open?.displayName ?? t("crew.title")}
        closeLabel={t("crew.close")}
        closeIcon={<Text>{"×"}</Text>}
      >
        <View style={{ gap: 12, paddingBottom: 12 }}>
          {open && isOwner && open.uid !== me ? (
            <>
              <View style={{ flexDirection: "row", gap: 8 }}>
                {(["editor", "viewer"] as const).map((r) => (
                  <Chip
                    key={r}
                    label={t(`trip.roles.${r}`)}
                    selected={open.role === r}
                    onPress={() => void setMemberRole(open, r)}
                  />
                ))}
              </View>
              <Button
                label={t("crew.remove")}
                variant="secondary"
                loading={busy}
                onPress={() => void removeMember(open)}
              />
            </>
          ) : null}
          {open && open.uid === me ? (
            <Button
              label={t("crew.leave")}
              variant="secondary"
              loading={busy}
              onPress={() => void removeMember(open)}
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

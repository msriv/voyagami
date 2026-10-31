import { useCallback, useEffect, useState } from "react";
import { Share, View } from "react-native";

import { ApiError, type components } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { Button, Card, Text } from "@/ui";

type Job = components["schemas"]["RecapVideo"];
type State = { kind: "loading" } | { kind: "off" } | { kind: "none" } | { kind: "job"; job: Job };

const POLL_MS = 3000;
const shareLink = (url: string) => Share.share({ message: url, url });

/** Make the recap as a vertical video and share it. Hidden while the server has no renderer (503). */
export function RecapVideo({ tripId, canEdit }: { tripId: string; canEdit: boolean }) {
  const { t } = useTranslation();
  const { api } = useAuth();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();

  const refresh = useCallback(async () => {
    try {
      const job = await api.unwrap(
        api.GET("/v1/trips/{tripId}/recap/video", { params: { path: { tripId } } }),
      );
      setState({ kind: "job", job });
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) setState({ kind: "none" });
      else if (e instanceof ApiError && e.status === 503) setState({ kind: "off" });
      // Any other failure keeps what is on screen; the next poll or visit tries again.
    }
  }, [api, tripId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const running =
    state.kind === "job" && (state.job.state === "queued" || state.job.state === "rendering");
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(() => void refresh(), POLL_MS);
    return () => clearInterval(timer);
  }, [running, refresh]);

  async function start() {
    setBusy(true);
    setError(undefined);
    try {
      const job = await api.unwrap(
        api.POST("/v1/trips/{tripId}/recap/video", { params: { path: { tripId } } }),
      );
      setState({ kind: "job", job });
    } catch (e) {
      if (e instanceof ApiError && e.status === 503) setState({ kind: "off" });
      else setError(t("recapVideo.startError"));
    } finally {
      setBusy(false);
    }
  }

  if (state.kind === "loading" || state.kind === "off") return null;
  if (state.kind === "none" && !canEdit) return null;

  const job = state.kind === "job" ? state.job : undefined;
  const limits = job
    ? [
        job.maxSeconds > 0
          ? t("recapVideo.length", { seconds: job.maxSeconds })
          : t("recapVideo.fullLength"),
        job.watermark ? t("recapVideo.watermark") : t("recapVideo.noWatermark"),
      ].join(" ")
    : t("recapVideo.freeNote");

  return (
    <Card style={{ gap: 8 }}>
      <Text variant="bodyStrong" heading>
        {t("recapVideo.title")}
      </Text>
      <Text variant="caption">{limits}</Text>
      <View accessibilityLiveRegion="polite" style={{ gap: 8 }}>
        {job && running ? <Text>{t("recapVideo.working", { percent: job.progress })}</Text> : null}
        {job?.state === "failed" ? <Text>{job.error ?? t("recapVideo.failed")}</Text> : null}
        {error ? <Text>{error}</Text> : null}
      </View>
      {job?.state === "ready" && job.url ? (
        <Button label={t("recapVideo.share")} onPress={() => void shareLink(job.url ?? "")} />
      ) : null}
      {canEdit && !running ? (
        <Button
          label={t(job ? "recapVideo.again" : "recapVideo.make")}
          variant={job?.state === "ready" ? "secondary" : "primary"}
          loading={busy}
          onPress={() => void start()}
        />
      ) : null}
    </Card>
  );
}

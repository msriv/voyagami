import { useCallback, useEffect, useState } from "react";
import { Pressable, View } from "react-native";

import { NetworkError } from "@voyagami/api-client";

import { useAuth } from "@/auth";
import { useTranslation } from "@/i18n";
import { useTheme } from "@/theme";
import { Button, IconButton, Input, Text } from "@/ui";

interface Task {
  id: string;
  title: string;
  done: boolean;
}

type State = { kind: "loading" } | { kind: "ready"; tasks: Task[] } | { kind: "error" };

const PENDING = "pending-";

/** A trip's to-dos. Ticking and deleting show at once and roll back if the server says no. */
export function TripTasks({ tripId, canEdit }: { tripId: string; canEdit: boolean }) {
  const { t } = useTranslation();
  const { colors, radius, hitArea } = useTheme();
  const { api } = useAuth();
  const [state, setState] = useState<State>({ kind: "loading" });
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState<string | undefined>();

  const load = useCallback(async () => {
    try {
      const { tasks } = await api.unwrap(
        api.GET("/v1/trips/{tripId}/tasks", { params: { path: { tripId } } }),
      );
      setState({ kind: "ready", tasks });
    } catch {
      setState({ kind: "error" });
    }
  }, [api, tripId]);

  useEffect(() => {
    void load();
  }, [load]);

  const tasks = state.kind === "ready" ? state.tasks : [];
  const put = (next: Task[]) => setState({ kind: "ready", tasks: next });
  const fail = (e: unknown) =>
    setMessage(t(e instanceof NetworkError ? "tasks.errors.offline" : "tasks.errors.generic"));

  async function toggle(task: Task) {
    if (task.id.startsWith(PENDING)) return;
    setMessage(undefined);
    const before = tasks;
    put(tasks.map((x) => (x.id === task.id ? { ...x, done: !x.done } : x)));
    try {
      await api.unwrap(
        api.PATCH("/v1/trips/{tripId}/tasks/{taskId}", {
          params: { path: { tripId, taskId: task.id } },
          body: { done: !task.done },
        }),
      );
    } catch (e) {
      put(before);
      fail(e);
    }
  }

  async function remove(task: Task) {
    if (task.id.startsWith(PENDING)) return;
    setMessage(undefined);
    const before = tasks;
    put(tasks.filter((x) => x.id !== task.id));
    try {
      await api.unwrap(
        api.DELETE("/v1/trips/{tripId}/tasks/{taskId}", {
          params: { path: { tripId, taskId: task.id } },
        }),
      );
    } catch (e) {
      put(before);
      fail(e);
    }
  }

  async function add() {
    const text = title.trim();
    if (!text) {
      setMessage(t("tasks.errors.blank"));
      return;
    }
    setMessage(undefined);
    setTitle("");
    const before = tasks;
    const temp = { id: `${PENDING}${Date.now()}`, title: text, done: false };
    put([...tasks, temp]);
    try {
      const saved = await api.unwrap(
        api.POST("/v1/trips/{tripId}/tasks", {
          params: { path: { tripId } },
          body: { title: text },
        }),
      );
      setState((s) =>
        s.kind === "ready"
          ? { kind: "ready", tasks: s.tasks.map((x) => (x.id === temp.id ? saved : x)) }
          : s,
      );
    } catch (e) {
      put(before);
      setTitle(text);
      fail(e);
    }
  }

  return (
    <View style={{ gap: 8 }}>
      <Text variant="title" heading>
        {t("tasks.heading")}
      </Text>
      {state.kind === "loading" ? <Text color="muted">{t("tasks.loading")}</Text> : null}
      {state.kind === "error" ? (
        <View style={{ gap: 8 }}>
          <Text>{t("tasks.loadError")}</Text>
          <Button label={t("tasks.retry")} variant="secondary" onPress={() => void load()} />
        </View>
      ) : null}
      {state.kind === "ready" && tasks.length === 0 ? (
        <Text color="muted">{t(canEdit ? "tasks.empty" : "tasks.emptyViewer")}</Text>
      ) : null}
      {tasks.map((task) => (
        <View
          key={task.id}
          style={{
            flexDirection: "row",
            alignItems: "center",
            gap: 8,
            backgroundColor: colors.white,
            borderRadius: radius.button,
          }}
        >
          <Pressable
            accessibilityRole="checkbox"
            accessibilityLabel={task.title}
            accessibilityState={{ checked: task.done, disabled: !canEdit }}
            disabled={!canEdit}
            onPress={() => void toggle(task)}
            style={{
              flex: 1,
              flexDirection: "row",
              alignItems: "center",
              gap: 12,
              minHeight: hitArea.min,
              paddingHorizontal: 12,
            }}
          >
            <View
              style={{
                width: 24,
                height: 24,
                borderRadius: 6,
                borderWidth: 2,
                borderColor: colors.indigo,
                backgroundColor: task.done ? colors.indigo : "transparent",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              {task.done ? <Text color="white">{"✓"}</Text> : null}
            </View>
            <Text
              color={task.done ? "muted" : "ink"}
              style={{ flex: 1, textDecorationLine: task.done ? "line-through" : "none" }}
            >
              {task.title}
            </Text>
          </Pressable>
          {canEdit ? (
            <IconButton
              accessibilityLabel={t("tasks.delete", { title: task.title })}
              onPress={() => void remove(task)}
            >
              <Text>{"×"}</Text>
            </IconButton>
          ) : null}
        </View>
      ))}
      {canEdit ? (
        <View style={{ gap: 8 }}>
          <Input
            label={t("tasks.newLabel")}
            value={title}
            onChangeText={setTitle}
            maxLength={120}
            returnKeyType="done"
            onSubmitEditing={() => void add()}
          />
          <Button label={t("tasks.add")} variant="secondary" onPress={() => void add()} />
        </View>
      ) : null}
      {message ? (
        <Text color="vermilion" accessibilityLiveRegion="polite">
          {message}
        </Text>
      ) : null}
    </View>
  );
}

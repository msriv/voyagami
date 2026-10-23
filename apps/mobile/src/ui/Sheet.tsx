import type { ReactNode } from "react";
import { Modal, Pressable, ScrollView, View } from "react-native";

import { useTheme } from "@/theme";

import { IconButton } from "./IconButton";
import { Text } from "./Text";
import { useReducedMotion } from "./useReducedMotion";

export interface SheetProps {
  visible: boolean;
  onClose: () => void;
  title: string;
  /** Screen reader label for the close button and the backdrop. */
  closeLabel: string;
  /** Visible glyph for the close button, hidden from screen readers. */
  closeIcon: ReactNode;
  children: ReactNode;
}

/**
 * A bottom sheet with a 30 radius top. It slides in, or just appears when reduce motion is on.
 * Content scrolls so it stays reachable at 200% text. Android back and the backdrop both close it.
 */
export function Sheet({ visible, onClose, title, closeLabel, closeIcon, children }: SheetProps) {
  const theme = useTheme();
  const reduceMotion = useReducedMotion();

  return (
    <Modal
      visible={visible}
      transparent
      animationType={reduceMotion ? "none" : "slide"}
      onRequestClose={onClose}
      accessibilityViewIsModal
    >
      <View style={{ flex: 1, justifyContent: "flex-end" }}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={closeLabel}
          onPress={onClose}
          style={{ flex: 1, backgroundColor: "rgba(20,22,27,0.45)" }}
        />
        <View
          style={{
            maxHeight: "85%",
            backgroundColor: theme.colors.washi,
            borderTopLeftRadius: theme.radius.sheetTop,
            borderTopRightRadius: theme.radius.sheetTop,
            paddingTop: 12,
            paddingHorizontal: 20,
            paddingBottom: 24,
          }}
        >
          <View
            style={{
              flexDirection: "row",
              alignItems: "center",
              justifyContent: "space-between",
              gap: 12,
            }}
          >
            <Text variant="title" heading style={{ flex: 1 }}>
              {title}
            </Text>
            <IconButton accessibilityLabel={closeLabel} onPress={onClose}>
              {closeIcon}
            </IconButton>
          </View>
          <ScrollView contentContainerStyle={{ paddingTop: 12, gap: 12 }}>{children}</ScrollView>
        </View>
      </View>
    </Modal>
  );
}

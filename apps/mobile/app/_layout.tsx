import { useFonts } from "expo-font";
import { Stack, useRouter } from "expo-router";
import * as SplashScreen from "expo-splash-screen";
import { StatusBar } from "expo-status-bar";
import { useEffect, useRef } from "react";

import { AuthProvider, useAuth } from "@/auth";
import "@/i18n";
import { ProfileProvider } from "@/profile";
import { fontAssets, ThemeProvider } from "@/theme";

void SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  const [fontsLoaded, fontError] = useFonts(fontAssets);
  const ready = fontsLoaded || fontError !== null;

  useEffect(() => {
    if (ready) void SplashScreen.hideAsync();
  }, [ready]);

  if (!ready) return null;

  return (
    <ThemeProvider>
      <AuthProvider>
        <ProfileProvider>
          <StatusBar style="dark" />
          <Screens />
        </ProfileProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

// Signed-in people only reach the tabs; everyone else only reaches sign-in. Nothing shows
// until the saved session has been read, so a returning person never sees sign-in flash by.
function Screens() {
  const { status } = useAuth();
  const router = useRouter();
  const previous = useRef(status);

  // When the session changes after launch, the screen we were on is now guarded off, so go to the
  // new start screen. The first read after launch is skipped so deep links keep working.
  useEffect(() => {
    if (previous.current !== "loading" && previous.current !== status) {
      router.replace("/");
    }
    previous.current = status;
  }, [status, router]);

  if (status === "loading") return null;
  const signedIn = status === "signedIn";
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Protected guard={signedIn}>
        <Stack.Screen name="(tabs)" />
        <Stack.Screen name="gallery" />
        <Stack.Screen name="onboarding/region" />
        <Stack.Screen name="onboarding/style" />
        <Stack.Screen name="onboarding/connect" />
        <Stack.Screen name="trips/new" options={{ presentation: "modal" }} />
        <Stack.Screen name="trips/[id]" />
        <Stack.Screen name="trips/[id]/itinerary" />
        <Stack.Screen name="trips/[id]/add-place" />
        <Stack.Screen name="trips/[id]/money" />
        <Stack.Screen name="trips/[id]/bookings" />
        <Stack.Screen name="trips/[id]/journal" />
        <Stack.Screen name="trips/[id]/crew" />
        <Stack.Screen name="trips/[id]/polls" />
        <Stack.Screen name="trips/[id]/live" />
        <Stack.Screen name="trips/[id]/settle" />
        <Stack.Screen name="join" />
        <Stack.Screen name="trips/[id]/add-booking" />
        <Stack.Screen name="trips/[id]/add-expense" />
      </Stack.Protected>
      <Stack.Protected guard={!signedIn}>
        <Stack.Screen name="(auth)" />
      </Stack.Protected>
    </Stack>
  );
}

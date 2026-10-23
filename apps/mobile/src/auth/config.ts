// shortcut: one dev default, set per build by EXPO_PUBLIC_API_URL; replace with per-environment config in P1.
// An Android emulator reaches the host at http://10.0.2.2:8080; a phone needs the computer's LAN address.
export const apiBaseUrl = process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:8080";

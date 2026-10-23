# Running the app on a phone or simulator

D phases use the **development build**: the Voyagami app with the Expo dev client inside. It loads your code from Metro on your computer, so you can change code and see it on the phone. Expo Go is not used because the app will add native modules (maps, push, purchases) that Expo Go does not include.

You need Node 22 and pnpm (see the [README](../README.md)). Run `make setup` first.

## Fastest: a simulator or emulator on your computer

No accounts and no EAS needed. The first run compiles the native app.

```sh
cd apps/mobile
npx expo run:ios       # macOS with Xcode
npx expo run:android   # Android Studio emulator, or a USB phone with developer mode on
```

After the first build, start only Metro with `pnpm --filter @voyagami/mobile start` and open the installed app.

`ios/` and `android/` are generated and ignored by git. Delete them to start clean.

## A physical phone with EAS Build

This needs a free [Expo account](https://expo.dev/signup) and, for an iPhone, an Apple developer account. Apple accounts are a D4 item (CLAUDE.md), so until then use the simulator, or an Android phone, which needs nothing from Google.

One-time setup, from `apps/mobile`:

```sh
npx eas-cli login
npx eas-cli init            # links the app to your Expo account and writes the project ID into app.json
```

Build, then install:

```sh
pnpm --filter @voyagami/mobile build:dev:android          # APK; open the link on the phone to install
pnpm --filter @voyagami/mobile build:dev:ios              # needs an Apple developer account and a registered device
pnpm --filter @voyagami/mobile build:dev:ios-simulator    # a build for the iOS simulator, no Apple account
```

`eas.json` has two profiles: `development` (internal distribution, dev client) and `development-simulator` (the same, for the iOS simulator). Production and store profiles come in P1/P2.

## Use it

1. Start Metro: `pnpm --filter @voyagami/mobile start`.
2. Open the Voyagami dev build on the phone. On a physical phone it must be on the same Wi-Fi as your computer; if it cannot connect, use `start --tunnel`.
3. Pick the server from the dev launcher, or scan the QR code in the terminal.

The dev build only needs rebuilding when native dependencies or `app.json` change. JavaScript changes reload on their own.

## If it does not work

- **"Unable to resolve module" after pulling:** run `make setup`, then restart Metro with `--clear`.
- **Phone cannot reach Metro:** same Wi-Fi, no VPN, or use `--tunnel`.
- **API calls from a phone:** `localhost` on the phone is the phone. Set `EXPO_PUBLIC_API_URL` before `make dev` (for example `EXPO_PUBLIC_API_URL=http://192.168.1.20:8080`) to your computer's LAN address, or to the dev Cloud Run URL. An Android emulator reaches your computer at `http://10.0.2.2:8080`. The iOS simulator can use the default `http://localhost:8080`.
- **Signing in against the local API:** run the API with `APP_ENV=dev` (code `123456`). Dev tokens stop working when the API restarts until `AUTH_SECRET` is set (VOY-211), so sign in again after a restart.

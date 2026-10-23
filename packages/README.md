# packages

Shared TypeScript packages. Each package extends `tsconfig.base.json` and the root `eslint.config.mjs`.

## api-client

Typed client for the API, generated from `api/openapi.yaml`.

```
make api-client      # regenerates the Go server and packages/api-client/src/generated/schema.d.ts
```

CI regenerates both and fails if the result differs from what is committed. Never edit `src/generated/`.

```ts
import { createApiClient, ApiError } from "@voyagami/api-client";

const api = createApiClient({
  baseUrl: "https://api.example",
  getTokens: () => session.read(), // { accessToken, refreshToken } from secure storage, or null
  setTokens: (t) => session.write(t), // called after a refresh, and with null when sign-in is needed again
  onSignedOut: () => router.replace("/sign-in"),
});

const me = await api.unwrap(api.GET("/v1/me")); // typed from the spec; throws ApiError on a non-2xx answer
```

What the client handles, so screens do not:

- **Auth header:** sent on every route except the sign-in ones (`/v1/auth/otp/*`, `/v1/auth/refresh`).
- **Token refresh:** a 401 triggers one refresh, shared by any requests that fail at the same time (a refresh token works once), then one retry. If the refresh fails the tokens are cleared and `onSignedOut` runs.
- **Errors:** `unwrap` throws `ApiError` with `status`, `code`, `requestId` and `fieldErrors`. Switch on `code`, never on `message`. A request that never reached the server throws `NetworkError`.
- **Offline hook:** pass `offline: { onNetworkError(request, error) }` to catch requests that could not reach the server. Return a `Response` to stand in for the answer (for example after queueing a write) or `undefined` to let it fail. The queue itself belongs to the Offline & Sync project (docs/sync.md).

The package is not wired into `apps/mobile` yet; add `"@voyagami/api-client": "workspace:*"` when the first screen needs it (sign-in, VOY-106).

import type { components } from "./generated/schema";

export type ApiErrorBody = components["schemas"]["Error"];

/** An error answer from the API (any non-2xx). `code` is stable; switch on it, not on `message`. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string | undefined;
  /** On `limit_reached`: the plan feature at its limit, like `bestOrder`. */
  readonly feature: string | undefined;
  readonly fieldErrors: readonly { field: string; message: string }[];

  constructor(status: number, body: Partial<ApiErrorBody> | undefined) {
    super(body?.message ?? `Request failed with status ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.code = body?.code ?? "unknown";
    this.requestId = body?.requestId;
    this.feature = body?.feature;
    this.fieldErrors = body?.fieldErrors ?? [];
  }
}

/** The request never got an answer: offline, DNS, timeout. */
export class NetworkError extends Error {
  constructor(cause: unknown) {
    super("The server could not be reached.", { cause });
    this.name = "NetworkError";
  }
}

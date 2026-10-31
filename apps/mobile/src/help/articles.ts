/** Help article ids, in the order they are shown. Words live in i18n under `help.articles.<id>`. */
export const HELP_ARTICLES = [
  "trial",
  "cancel",
  "restore",
  "tripPass",
  "invite",
  "join",
  "roles",
  "settleUp",
  "currencies",
  "offline",
  "sync",
  "location",
  "notifications",
  "picks",
  "stays",
] as const;

export type HelpArticleId = (typeof HELP_ARTICLES)[number];

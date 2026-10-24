// Shared ESLint flat config. Workspaces extend this and add their own rules
// (for example React Native rules in apps/mobile).
import js from "@eslint/js";
import prettier from "eslint-config-prettier";
import react from "eslint-plugin-react";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    ignores: [
      "**/node_modules/**",
      "**/dist/**",
      "**/build/**",
      "**/coverage/**",
      "**/.expo/**",
      "design/**",
      "packages/api-client/src/generated/**",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.strict,
  prettier,
  {
    // User-facing text must come from i18n (CLAUDE.md), so no literal strings in JSX.
    files: ["apps/mobile/**/*.tsx"],
    plugins: { react },
    rules: { "react/jsx-no-literals": "error" },
  },
  {
    // Nothing is hard-coded to one country (CLAUDE.md): money, dates, distances and phone numbers
    // are shown only through @voyagami/locale. The locale module and tests are exempt.
    files: ["apps/mobile/**/*.{ts,tsx}", "packages/*/src/**/*.ts"],
    ignores: [
      "packages/locale/**",
      "packages/api-client/src/generated/**",
      "**/__tests__/**",
      "**/*.test.{ts,tsx}",
    ],
    rules: {
      "no-restricted-properties": [
        "error",
        ...["toFixed", "toLocaleString", "toLocaleDateString", "toLocaleTimeString"].map(
          (property) => ({
            property,
            message: "Format numbers, money and dates with @voyagami/locale, not by hand.",
          }),
        ),
        ...["NumberFormat", "DateTimeFormat", "RelativeTimeFormat"].map((property) => ({
          object: "Intl",
          property,
          message: "Format numbers, money and dates with @voyagami/locale, not Intl directly.",
        })),
      ],
      "no-restricted-syntax": [
        "error",
        ...[
          ["a currency symbol", "[₹€£¥₫₩₽₺฿]"],
          ["a dollar amount", "\\$\\s?\\d|\\d\\s?\\$"],
          ["a distance unit", "(^|\\d)\\s?(km|mi|kilometers|miles)$"],
          ["a calling code", "^\\+\\d{1,3}(\\s|$)"],
          ["a date format", "\\b(YYYY|yyyy|DD|MMM?)\\b"],
        ].flatMap(([what, pattern]) => [
          {
            selector: `Literal[value=/${pattern}/]`,
            message: `Hard-coded ${what}. Use @voyagami/locale (formatMoney, formatDate, formatDistance, toE164) or i18n.`,
          },
          {
            selector: `TemplateElement[value.raw=/${pattern}/]`,
            message: `Hard-coded ${what}. Use @voyagami/locale (formatMoney, formatDate, formatDistance, toE164) or i18n.`,
          },
        ]),
      ],
    },
  },
  {
    // Tests may render literal text.
    files: ["**/__tests__/**"],
    rules: { "react/jsx-no-literals": "off" },
  },
  {
    // Metro, Babel and Jest read these as CommonJS.
    files: ["**/*.config.js"],
    languageOptions: { globals: { module: "writable" } },
    rules: { "@typescript-eslint/no-require-imports": "off" },
  },
);

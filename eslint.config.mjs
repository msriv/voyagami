// Shared ESLint flat config. Workspaces extend this and add their own rules
// (for example React Native rules in apps/mobile).
import js from "@eslint/js";
import prettier from "eslint-config-prettier";
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
);

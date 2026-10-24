module.exports = {
  preset: "jest-expo",
  setupFilesAfterEnv: ["<rootDir>/test-utils/setup.ts"],
  moduleNameMapper: { "^@/(.*)$": "<rootDir>/src/$1" },
};

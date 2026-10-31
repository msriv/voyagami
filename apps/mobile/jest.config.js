module.exports = {
  preset: "jest-expo",
  setupFilesAfterEnv: ["<rootDir>/test-utils/setup.tsx"],
  moduleNameMapper: { "^@/(.*)$": "<rootDir>/src/$1" },
};

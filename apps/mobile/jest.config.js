module.exports = {
  preset: "jest-expo",
  setupFilesAfterEnv: ["<rootDir>/test-utils/setup.tsx"],
  moduleNameMapper: { "^@/(.*)$": "<rootDir>/src/$1" },
  // Screen tests render the whole route tree; a cold CI runner needs more than jest's 5s.
  testTimeout: 20000,
};

// Native permission modules have no JS implementation under Jest.
jest.mock("expo-contacts", () => ({
  requestPermissionsAsync: jest.fn(() => Promise.resolve({ granted: true })),
}));
jest.mock("expo-notifications", () => ({
  requestPermissionsAsync: jest.fn(() => Promise.resolve({ granted: false })),
}));

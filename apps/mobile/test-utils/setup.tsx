// Native permission modules have no JS implementation under Jest.
jest.mock("expo-contacts", () => ({
  requestPermissionsAsync: jest.fn(() => Promise.resolve({ granted: true })),
}));
jest.mock("expo-notifications", () => ({
  requestPermissionsAsync: jest.fn(() => Promise.resolve({ granted: false })),
}));

// The native map has no JS implementation; screens only need it to render children.
jest.mock("react-native-maps", () => {
  const { View } = jest.requireActual<typeof import("react-native")>("react-native");
  const MapView = ({ children }: { children?: unknown }) => (
    <View testID="map">{children as never}</View>
  );
  return { __esModule: true, default: MapView, Marker: () => null, Polyline: () => null };
});

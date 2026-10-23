import { createContext, useContext, useMemo, type ReactNode } from "react";

import { fontFamily } from "./fonts";
import { hitArea, lightPalette, radius, type Palette } from "./tokens";

export interface Theme {
  colors: Palette;
  fontFamily: typeof fontFamily;
  radius: typeof radius;
  hitArea: typeof hitArea;
}

const lightTheme: Theme = { colors: lightPalette, fontFamily, radius, hitArea };

const ThemeContext = createContext<Theme>(lightTheme);

// Light mode only for now. Dark switches on the colour scheme once its tokens exist.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useMemo(() => lightTheme, []);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

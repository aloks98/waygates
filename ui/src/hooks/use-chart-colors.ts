import { useTheme } from 'next-themes';
import { useEffect, useState } from 'react';

// CSS custom properties (hex literals in app.css, theme-dependent) that charts
// need as concrete color values. Canvas-based charts build gradients via
// CanvasGradient.addColorStop(), which cannot parse `var(--x)` — so any color
// passed into a chart series must be resolved to its computed value first.
const CHART_TOKENS = [
  '--chart-1',
  '--chart-2',
  '--chart-3',
  '--chart-4',
  '--chart-5',
  '--destructive',
] as const;

export type ChartToken = (typeof CHART_TOKENS)[number];

function resolveTokens(): Record<ChartToken, string> {
  const out = {} as Record<ChartToken, string>;
  if (typeof window === 'undefined') return out;
  const style = getComputedStyle(document.documentElement);
  for (const token of CHART_TOKENS) {
    out[token] = style.getPropertyValue(token).trim();
  }
  return out;
}

/**
 * Resolves chart color CSS variables to concrete hex values for the active
 * theme. Re-resolves whenever the theme changes so charts re-color on
 * light/dark toggle.
 */
export function useChartColors(): Record<ChartToken, string> {
  const { resolvedTheme } = useTheme();
  const [colors, setColors] = useState<Record<ChartToken, string>>(resolveTokens);

  useEffect(() => {
    setColors(resolveTokens());
  }, [resolvedTheme]);

  return colors;
}

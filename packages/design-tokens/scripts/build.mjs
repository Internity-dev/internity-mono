// Reads ../tokens.json and emits ../dist/theme.css — a Tailwind v4 + shadcn-vue
// compatible theme file (raw color ramps, semantic vars for light/dark, and an
// `@theme inline` block that turns both into Tailwind utility classes).
// Run via `pnpm --filter @internity/design-tokens build` (or `pnpm tokens:build` at root).
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const tokens = JSON.parse(readFileSync(join(root, "tokens.json"), "utf-8"));

function resolveColor(ref) {
  if (typeof ref !== "string") throw new Error(`Expected color reference string, got ${JSON.stringify(ref)}`);
  if (ref.startsWith("#")) return ref;
  const [group, shade] = ref.split(".");
  if (shade) {
    const value = tokens.palette[group]?.[shade];
    if (!value) throw new Error(`Unknown palette reference: ${ref}`);
    return value;
  }
  const value = tokens.palette[group];
  if (!value) throw new Error(`Unknown palette reference: ${ref}`);
  return value;
}

function kebab(key) {
  return key.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase();
}

const rampGroups = ["primary", "accent", "neutral"];
const flatColors = ["success", "warning", "danger", "info"];

function renderRampVars() {
  const lines = [];
  for (const group of rampGroups) {
    for (const [shade, hex] of Object.entries(tokens.palette[group])) {
      lines.push(`  --color-${group}-${shade}: ${hex};`);
    }
  }
  for (const key of flatColors) {
    lines.push(`  --color-${key}: ${resolveColor(key)};`);
  }
  return lines.join("\n");
}

function renderSemanticVars(mode) {
  const entries = tokens.semantic[mode];
  return Object.entries(entries)
    .map(([key, ref]) => `  --${kebab(key)}: ${resolveColor(ref)};`)
    .join("\n");
}

function renderRadiusVars() {
  return `  --radius: ${tokens.radius.base};`;
}

function renderThemeInline() {
  const lines = [];
  for (const group of rampGroups) {
    for (const shade of Object.keys(tokens.palette[group])) {
      lines.push(`  --color-${group}-${shade}: var(--color-${group}-${shade});`);
    }
  }
  for (const key of flatColors) lines.push(`  --color-${key}: var(--color-${key});`);
  for (const key of Object.keys(tokens.semantic.light)) {
    lines.push(`  --color-${kebab(key)}: var(--${kebab(key)});`);
  }
  lines.push(`  --radius-sm: calc(var(--radius) - 4px);`);
  lines.push(`  --radius-md: calc(var(--radius) - 2px);`);
  lines.push(`  --radius-lg: var(--radius);`);
  lines.push(`  --radius-xl: calc(var(--radius) + 4px);`);
  lines.push(`  --font-sans: ${tokens.typography.sans.map((f) => (f.includes(" ") ? `"${f}"` : f)).join(", ")};`);
  lines.push(`  --font-display: ${tokens.typography.display.map((f) => (f.includes(" ") ? `"${f}"` : f)).join(", ")};`);
  lines.push(`  --font-mono: ${tokens.typography.mono.map((f) => (f.includes(" ") ? `"${f}"` : f)).join(", ")};`);
  return lines.join("\n");
}

const css = `/* GENERATED FILE — do not edit by hand.
 * Source: packages/design-tokens/tokens.json
 * Regenerate: pnpm tokens:build
 */

:root {
${renderRampVars()}
${renderRadiusVars()}
${renderSemanticVars("light")}
}

.dark {
${renderSemanticVars("dark")}
}

@theme inline {
${renderThemeInline()}
}
`;

mkdirSync(join(root, "dist"), { recursive: true });
writeFileSync(join(root, "dist", "theme.css"), css);
console.log("Wrote packages/design-tokens/dist/theme.css");

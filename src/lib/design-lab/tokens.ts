export type ColorTokenSpec = {
  name: string
  cssVar: string
  lightValue: string
  darkValue: string
  usage: string
  onLightValue?: string
  onDarkValue?: string
}

export type TypographyTokenSpec = {
  name: string
  fontSize: string
  lineHeight: string
  fontWeight: string
  preview: string
}

export type ScaleTokenSpec = {
  name: string
  cssVar: string
  value: string
}

export const semanticColorTokens: ColorTokenSpec[] = [
  {
    name: "Primary",
    cssVar: "--primary",
    lightValue: "#FE628E",
    darkValue: "#FE628E",
    usage: "Primary action, selected state, brand emphasis",
    onLightValue: "#1D0910",
    onDarkValue: "#1D0910",
  },
  {
    name: "Success",
    cssVar: "--success",
    lightValue: "#2F9E78",
    darkValue: "#8FD6BF",
    usage: "Success badges, positive status, completed actions",
    onLightValue: "#071B14",
    onDarkValue: "#08140F",
  },
  {
    name: "Warning",
    cssVar: "--warning",
    lightValue: "#D89A1B",
    darkValue: "#F5B971",
    usage: "Caution states, warnings, pending confirmations",
    onLightValue: "#231703",
    onDarkValue: "#2B1600",
  },
  {
    name: "Danger",
    cssVar: "--danger",
    lightValue: "#E14B6D",
    darkValue: "#FF6F87",
    usage: "Destructive actions and critical alerts",
    onLightValue: "#25070F",
    onDarkValue: "#2B0811",
  },
  {
    name: "Info",
    cssVar: "--info",
    lightValue: "#5B6FD4",
    darkValue: "#8B9CFF",
    usage: "Informational callouts and secondary highlight states",
    onLightValue: "#0B1332",
    onDarkValue: "#0D1330",
  },
]

export const neutralColorTokens: ColorTokenSpec[] = [
  {
    name: "Background",
    cssVar: "--background",
    lightValue: "#F4F6FC",
    darkValue: "#0D0F1A",
    usage: "App canvas background",
  },
  {
    name: "Surface",
    cssVar: "--surface",
    lightValue: "#FFFFFF",
    darkValue: "#141826",
    usage: "Primary card and panel surface",
  },
  {
    name: "Surface Elevated",
    cssVar: "--surface-elevated",
    lightValue: "#FFFFFF",
    darkValue: "#171B2B",
    usage: "Floating panels and raised containers",
  },
  {
    name: "Surface Muted",
    cssVar: "--surface-muted",
    lightValue: "#EBEEF5",
    darkValue: "#121827",
    usage: "Sub-panels, secondary fills, muted strips",
  },
  {
    name: "Foreground",
    cssVar: "--foreground",
    lightValue: "#0F1219",
    darkValue: "#F8F7FB",
    usage: "Primary text content",
  },
]

export const typographyTokens: TypographyTokenSpec[] = [
  { name: "H1", fontSize: "32px", lineHeight: "40px", fontWeight: "700", preview: "Curated Design Lab" },
  { name: "H2", fontSize: "28px", lineHeight: "36px", fontWeight: "700", preview: "Token System" },
  { name: "H3", fontSize: "24px", lineHeight: "32px", fontWeight: "600", preview: "Component States" },
  { name: "H4", fontSize: "20px", lineHeight: "28px", fontWeight: "600", preview: "Interactive Playground" },
  { name: "Body", fontSize: "14px", lineHeight: "24px", fontWeight: "400", preview: "Use semantic tokens and existing UI primitives before inventing new styles." },
  { name: "Caption", fontSize: "12px", lineHeight: "18px", fontWeight: "500", preview: "Token metadata, helper copy, and support notes." },
]

export const radiusTokens: ScaleTokenSpec[] = [
  { name: "None", cssVar: "--radius-none", value: "0px" },
  { name: "XS", cssVar: "--radius-xs", value: "4px" },
  { name: "SM", cssVar: "--radius-sm", value: "6px" },
  { name: "MD", cssVar: "--radius-md", value: "10px" },
  { name: "LG", cssVar: "--radius-lg", value: "14px" },
  { name: "XL", cssVar: "--radius-xl", value: "18px" },
  { name: "Full", cssVar: "--radius-full", value: "9999px" },
]

export const shadowTokens: ScaleTokenSpec[] = [
  { name: "Flat", cssVar: "--shadow-flat", value: "none" },
  { name: "SM", cssVar: "--shadow-sm", value: "0 1px 2px rgb(15 18 25 / 6%)" },
  { name: "MD", cssVar: "--shadow-md", value: "0 8px 24px rgb(15 18 25 / 8%)" },
  { name: "LG", cssVar: "--shadow-lg", value: "0 16px 40px rgb(15 18 25 / 12%)" },
  { name: "XL", cssVar: "--shadow-xl", value: "0 24px 60px rgb(15 18 25 / 16%)" },
]

export const spacingTokens: ScaleTokenSpec[] = [
  { name: "1", cssVar: "--space-1", value: "4px" },
  { name: "2", cssVar: "--space-2", value: "8px" },
  { name: "3", cssVar: "--space-3", value: "12px" },
  { name: "4", cssVar: "--space-4", value: "16px" },
  { name: "5", cssVar: "--space-5", value: "20px" },
  { name: "6", cssVar: "--space-6", value: "24px" },
  { name: "8", cssVar: "--space-8", value: "32px" },
  { name: "10", cssVar: "--space-10", value: "40px" },
  { name: "12", cssVar: "--space-12", value: "48px" },
  { name: "16", cssVar: "--space-16", value: "64px" },
]

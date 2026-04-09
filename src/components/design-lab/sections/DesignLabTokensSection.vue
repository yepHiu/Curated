<script setup lang="ts">
import { computed } from "vue"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { getContrastRatio } from "@/lib/design-lab/contrast"
import {
  neutralColorTokens,
  radiusTokens,
  semanticColorTokens,
  shadowTokens,
  spacingTokens,
  typographyTokens,
  type ColorTokenSpec,
} from "@/lib/design-lab/tokens"

type ColorRow = ColorTokenSpec & {
  contrastLight: number | null
  contrastDark: number | null
}

function withContrast(tokens: ColorTokenSpec[]) {
  return tokens.map((token): ColorRow => ({
    ...token,
    contrastLight:
      token.onLightValue ? getContrastRatio(token.onLightValue, token.lightValue) : null,
    contrastDark:
      token.onDarkValue ? getContrastRatio(token.onDarkValue, token.darkValue) : null,
  }))
}

function getContrastGrade(ratio: number | null): string {
  if (ratio == null) return "N/A"
  if (ratio >= 7) return "AAA"
  if (ratio >= 4.5) return "AA"
  return "Fail"
}

const semanticRows = computed(() => withContrast(semanticColorTokens))
const neutralRows = computed(() => withContrast(neutralColorTokens))
</script>

<template>
  <section id="tokens" class="scroll-mt-24">
    <div class="flex flex-col gap-6">
      <Card class="rounded-2xl border border-border bg-card shadow-sm">
        <CardHeader>
          <CardTitle>Tokens</CardTitle>
          <CardDescription>
            Design tokens define the visual language for light and dark mode before component-level decisions are layered on top.
          </CardDescription>
        </CardHeader>
      </Card>

      <Card class="rounded-2xl border border-border bg-card shadow-sm">
        <CardHeader>
          <CardTitle>Semantic Colors</CardTitle>
          <CardDescription>
            Curated keeps the current brand core and extends support colors for success, warning, danger, and info.
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-4 lg:grid-cols-2">
          <article
            v-for="token in semanticRows"
            :key="token.cssVar"
            class="rounded-2xl border border-border/70 bg-surface p-4"
          >
            <div class="flex items-start justify-between gap-4">
              <div>
                <h3 class="text-sm font-semibold text-foreground">{{ token.name }}</h3>
                <p class="mt-1 font-mono text-xs text-muted-foreground">{{ token.cssVar }}</p>
              </div>
              <Badge variant="outline" class="rounded-full px-3 py-1 text-[11px] uppercase tracking-[0.14em]">
                semantic
              </Badge>
            </div>

            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              <div class="rounded-xl border border-border/70 p-3">
                <div class="mb-3 flex items-center justify-between">
                  <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Light</span>
                  <span class="font-mono text-xs text-foreground">{{ token.lightValue }}</span>
                </div>
                <div class="flex h-16 items-center justify-center rounded-xl" :style="{ backgroundColor: token.lightValue, color: token.onLightValue ?? '#0F1219' }">
                  Aa
                </div>
                <p class="mt-3 text-xs text-muted-foreground">
                  Contrast:
                  <span class="font-medium text-foreground">
                    {{ token.contrastLight ?? "N/A" }}
                  </span>
                  <span v-if="token.contrastLight !== null">({{ getContrastGrade(token.contrastLight) }})</span>
                </p>
              </div>

              <div class="rounded-xl border border-border/70 p-3">
                <div class="mb-3 flex items-center justify-between">
                  <span class="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">Dark</span>
                  <span class="font-mono text-xs text-foreground">{{ token.darkValue }}</span>
                </div>
                <div class="flex h-16 items-center justify-center rounded-xl" :style="{ backgroundColor: token.darkValue, color: token.onDarkValue ?? '#F8F7FB' }">
                  Aa
                </div>
                <p class="mt-3 text-xs text-muted-foreground">
                  Contrast:
                  <span class="font-medium text-foreground">
                    {{ token.contrastDark ?? "N/A" }}
                  </span>
                  <span v-if="token.contrastDark !== null">({{ getContrastGrade(token.contrastDark) }})</span>
                </p>
              </div>
            </div>

            <p class="mt-4 text-sm leading-6 text-muted-foreground">{{ token.usage }}</p>
          </article>
        </CardContent>
      </Card>

      <Card class="rounded-2xl border border-border bg-card shadow-sm">
        <CardHeader>
          <CardTitle>Neutral Surfaces</CardTitle>
          <CardDescription>
            Surface and foreground tokens define the density and depth of the application shell.
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-4 lg:grid-cols-2">
          <article
            v-for="token in neutralRows"
            :key="token.cssVar"
            class="rounded-2xl border border-border/70 bg-surface p-4"
          >
            <div class="flex items-start justify-between gap-4">
              <div>
                <h3 class="text-sm font-semibold text-foreground">{{ token.name }}</h3>
                <p class="mt-1 font-mono text-xs text-muted-foreground">{{ token.cssVar }}</p>
              </div>
              <span class="font-mono text-xs text-foreground">{{ token.lightValue }} / {{ token.darkValue }}</span>
            </div>
            <p class="mt-4 text-sm leading-6 text-muted-foreground">{{ token.usage }}</p>
          </article>
        </CardContent>
      </Card>

      <div class="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <Card class="rounded-2xl border border-border bg-card shadow-sm">
          <CardHeader>
            <CardTitle>Typography</CardTitle>
            <CardDescription>
              Heading and body styles should be stable enough to power generated examples and component demos.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div
              v-for="token in typographyTokens"
              :key="token.name"
              class="rounded-2xl border border-border/70 p-4"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="text-sm font-semibold text-foreground">{{ token.name }}</div>
                <div class="font-mono text-xs text-muted-foreground">
                  {{ token.fontSize }} / {{ token.lineHeight }} / {{ token.fontWeight }}
                </div>
              </div>
              <p
                class="mt-3 text-foreground"
                :style="{
                  fontSize: token.fontSize,
                  lineHeight: token.lineHeight,
                  fontWeight: token.fontWeight,
                }"
              >
                {{ token.preview }}
              </p>
            </div>
          </CardContent>
        </Card>

        <div class="grid gap-6">
          <Card class="rounded-2xl border border-border bg-card shadow-sm">
            <CardHeader>
              <CardTitle>Radius</CardTitle>
              <CardDescription>Rounded corners scale from utility chips to large cards.</CardDescription>
            </CardHeader>
            <CardContent class="grid gap-3">
              <div
                v-for="token in radiusTokens"
                :key="token.cssVar"
                class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 p-3"
              >
                <div class="flex items-center gap-3">
                  <div class="size-10 border border-border/70 bg-muted/60" :style="{ borderRadius: token.value }" />
                  <div>
                    <p class="text-sm font-medium text-foreground">{{ token.name }}</p>
                    <p class="font-mono text-xs text-muted-foreground">{{ token.cssVar }}</p>
                  </div>
                </div>
                <span class="font-mono text-xs text-foreground">{{ token.value }}</span>
              </div>
            </CardContent>
          </Card>

          <Card class="rounded-2xl border border-border bg-card shadow-sm">
            <CardHeader>
              <CardTitle>Shadow And Spacing</CardTitle>
              <CardDescription>Elevation and spacing scale should stay consistent across settings, cards, and overlays.</CardDescription>
            </CardHeader>
            <CardContent class="grid gap-4">
              <div class="grid gap-3">
                <div
                  v-for="token in shadowTokens"
                  :key="token.cssVar"
                  class="flex items-center justify-between gap-4 rounded-2xl border border-border/70 p-3"
                >
                  <div>
                    <p class="text-sm font-medium text-foreground">{{ token.name }}</p>
                    <p class="font-mono text-xs text-muted-foreground">{{ token.cssVar }}</p>
                  </div>
                  <div class="h-12 w-20 rounded-xl bg-card" :style="{ boxShadow: token.value }" />
                </div>
              </div>

              <div class="grid gap-2">
                <div
                  v-for="token in spacingTokens"
                  :key="token.cssVar"
                  class="flex items-center gap-3 rounded-2xl border border-border/70 p-3"
                >
                  <div class="w-12 font-mono text-xs text-muted-foreground">{{ token.cssVar }}</div>
                  <div class="w-8 text-sm font-medium text-foreground">{{ token.name }}</div>
                  <div class="h-2 rounded-full bg-primary/75" :style="{ width: token.value }" />
                  <div class="font-mono text-xs text-foreground">{{ token.value }}</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  </section>
</template>

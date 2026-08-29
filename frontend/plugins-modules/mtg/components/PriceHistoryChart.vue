<script setup lang="ts">
  import { computed, ref, onMounted } from "vue";
  import type { PriceHistoryEntry } from "~~/lib/api/types/data-contracts";
  import { useFormatCurrency } from "~/composables/use-formatters";

  const props = defineProps<{
    entries: PriceHistoryEntry[];
    purchasePrice?: number;
    purchaseDate?: string | Date;
  }>();

  const fmtCurrency = ref<((v: number | string) => string) | null>(null);
  onMounted(async () => {
    fmtCurrency.value = await useFormatCurrency();
  });

  const formatCurrency = (val: number | string) => {
    if (fmtCurrency.value) {
      return fmtCurrency.value(val);
    }
    return `$${Number(val || 0).toFixed(2)}`;
  };

  const selectedRange = ref<"1M" | "3M" | "6M" | "1Y" | "ALL">("ALL");

  const filteredEntries = computed(() => {
    if (!props.entries || props.entries.length === 0) {
      return [];
    }

    const sorted = [...props.entries].sort(
      (a, b) => new Date(a.recordedAt).getTime() - new Date(b.recordedAt).getTime()
    );

    if (selectedRange.value === "ALL") {
      return sorted;
    }

    const now = new Date().getTime();
    const ranges: Record<string, number> = {
      "1M": 30 * 24 * 60 * 60 * 1000,
      "3M": 90 * 24 * 60 * 60 * 1000,
      "6M": 180 * 24 * 60 * 60 * 1000,
      "1Y": 365 * 24 * 60 * 60 * 1000,
    };

    const rangeMs = ranges[selectedRange.value];
    if (!rangeMs) return sorted;
    const cutoff = now - rangeMs;
    const inRange = sorted.filter(e => new Date(e.recordedAt).getTime() >= cutoff);
    return inRange.length > 0 ? inRange : sorted;
  });

  // SVG Chart Dimensions
  const svgWidth = 600;
  const svgHeight = 220;
  const padding = { top: 20, right: 30, bottom: 35, left: 55 };

  const chartWidth = computed(() => svgWidth - padding.left - padding.right);
  const chartHeight = computed(() => svgHeight - padding.top - padding.bottom);

  const priceRange = computed(() => {
    if (filteredEntries.value.length === 0) return { min: 0, max: 100 };
    const prices = filteredEntries.value.map(e => e.price);
    if (props.purchasePrice && props.purchasePrice > 0) {
      prices.push(props.purchasePrice);
    }
    const min = Math.min(...prices);
    const max = Math.max(...prices);
    const buffer = (max - min) * 0.1 || 10;
    return {
      min: Math.max(0, min - buffer),
      max: max + buffer,
    };
  });

  const getY = (price: number) => {
    const range = priceRange.value.max - priceRange.value.min;
    if (range === 0) return padding.top + chartHeight.value / 2;
    const normalized = (price - priceRange.value.min) / range;
    return padding.top + chartHeight.value - normalized * chartHeight.value;
  };

  const getX = (recordedAt: string | Date) => {
    if (filteredEntries.value.length <= 1) return padding.left + chartWidth.value / 2;
    const firstTime = new Date(filteredEntries.value[0]?.recordedAt ?? 0).getTime();
    const lastTime = new Date(filteredEntries.value[filteredEntries.value.length - 1]?.recordedAt ?? 0).getTime();
    const timeRange = lastTime - firstTime;
    if (timeRange === 0) return padding.left + chartWidth.value / 2;
    const currentTime = new Date(recordedAt).getTime();
    return padding.left + ((currentTime - firstTime) / timeRange) * chartWidth.value;
  };

  const points = computed<{ x: number; y: number; entry: PriceHistoryEntry }[]>(() => {
    return filteredEntries.value.map(entry => ({
      x: getX(entry.recordedAt),
      y: getY(entry.price),
      entry,
    }));
  });

  const linePath = computed(() => {
    if (points.value.length < 2) return "";
    return points.value.reduce((acc, pt, i) => {
      return i === 0 ? `M ${pt.x} ${pt.y}` : `${acc} L ${pt.x} ${pt.y}`;
    }, "");
  });

  const areaPath = computed(() => {
    if (points.value.length < 2) return "";
    const first = points.value[0];
    const last = points.value[points.value.length - 1];
    const baselineY = padding.top + chartHeight.value;
    if (!first || !last) return "";
    return `${linePath.value} L ${last.x} ${baselineY} L ${first.x} ${baselineY} Z`;
  });

  const purchaseLineY = computed(() => {
    if (!props.purchasePrice || props.purchasePrice <= 0) return null;
    return getY(props.purchasePrice);
  });

  const yTicks = computed(() => {
    const ticks: { value: number; y: number; label: string }[] = [];
    const count = 4;
    const step = (priceRange.value.max - priceRange.value.min) / (count - 1);
    for (let i = 0; i < count; i++) {
      const val = priceRange.value.min + step * i;
      ticks.push({
        value: val,
        y: getY(val),
        label: formatCurrency(val),
      });
    }
    return ticks;
  });

  const xTicks = computed(() => {
    if (filteredEntries.value.length === 0) return [];
    const ticks: { x: number; label: string }[] = [];
    const count = Math.min(4, filteredEntries.value.length);
    const step = Math.floor(filteredEntries.value.length / (count - 1 || 1));

    for (let i = 0; i < filteredEntries.value.length; i += step) {
      const entry = filteredEntries.value[i];
      if (entry) {
        const d = new Date(entry.recordedAt);
        ticks.push({
          x: getX(entry.recordedAt),
          label: d.toLocaleDateString(undefined, { month: "short", day: "numeric" }),
        });
      }
    }

    // Ensure the last entry is included
    const lastEntry = filteredEntries.value[filteredEntries.value.length - 1];
    if (lastEntry && !ticks.some(t => Math.abs(t.x - getX(lastEntry.recordedAt)) < 20)) {
      const d = new Date(lastEntry.recordedAt);
      ticks.push({
        x: getX(lastEntry.recordedAt),
        label: d.toLocaleDateString(undefined, { month: "short", day: "numeric" }),
      });
    }

    return ticks;
  });

  // Tooltip state
  const hoveredPoint = ref<{ x: number; y: number; entry: PriceHistoryEntry } | null>(null);

  const isPositiveTrend = computed(() => {
    if (points.value.length < 2) return true;
    const first = points.value[0]?.entry.price ?? 0;
    const last = points.value[points.value.length - 1]?.entry.price ?? 0;
    return last >= first;
  });
</script>

<template>
  <div class="flex flex-col space-y-3">
    <!-- Header & Range Selector -->
    <div class="flex items-center justify-between">
      <div class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {{ $t("components.item.price_chart.title") }}
      </div>
      <div class="flex items-center space-x-1 rounded-lg bg-muted/60 p-0.5">
        <button
          v-for="r in ['1M', '3M', '6M', '1Y', 'ALL'] as const"
          :key="r"
          type="button"
          class="rounded px-2 py-0.5 text-xs font-medium transition-all"
          :class="
            selectedRange === r
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          "
          @click="selectedRange = r"
        >
          {{ r }}
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-if="filteredEntries.length === 0"
      class="flex h-44 flex-col items-center justify-center rounded-lg border border-dashed text-muted-foreground"
    >
      <p class="text-sm">{{ $t("components.item.price_chart.no_data") }}</p>
      <p class="text-xs text-muted-foreground/80">
        {{ $t("components.item.price_chart.no_data_hint") }}
      </p>
    </div>

    <!-- Single Point State -->
    <div
      v-else-if="filteredEntries.length === 1"
      class="flex h-44 flex-col items-center justify-center rounded-lg border bg-muted/20 p-4 text-center"
    >
      <div class="text-2xl font-bold tracking-tight text-foreground">
        {{ formatCurrency(filteredEntries[0]?.price ?? 0) }}
      </div>
      <p class="mt-1 text-xs text-muted-foreground">
        {{
          $t("components.item.price_chart.single_point", {
            date: new Date(filteredEntries[0]?.recordedAt ?? 0).toLocaleDateString(),
          })
        }}
      </p>
    </div>

    <!-- Interactive SVG Chart -->
    <div v-else class="relative w-full overflow-hidden">
      <svg
        :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
        class="h-auto w-full overflow-visible"
        @mouseleave="hoveredPoint = null"
      >
        <defs>
          <linearGradient id="priceGradientPositive" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#10b981" stop-opacity="0.35" />
            <stop offset="100%" stop-color="#10b981" stop-opacity="0.0" />
          </linearGradient>
          <linearGradient id="priceGradientNegative" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="#f43f5e" stop-opacity="0.35" />
            <stop offset="100%" stop-color="#f43f5e" stop-opacity="0.0" />
          </linearGradient>
        </defs>

        <!-- Horizontal Grid Lines & Y-Labels -->
        <g class="grid-lines">
          <template v-for="tick in yTicks" :key="tick.y">
            <line
              :x1="padding.left"
              :y1="tick.y"
              :x2="svgWidth - padding.right"
              :y2="tick.y"
              stroke="currentColor"
              class="text-border/60"
              stroke-width="1"
              stroke-dasharray="3 3"
            />
            <text :x="padding.left - 8" :y="tick.y + 3.5" text-anchor="end" class="fill-muted-foreground text-[10px]">
              {{ tick.label }}
            </text>
          </template>
        </g>

        <!-- X-Axis Labels -->
        <g class="x-labels">
          <text
            v-for="(tick, idx) in xTicks"
            :key="idx"
            :x="tick.x"
            :y="svgHeight - 10"
            text-anchor="middle"
            class="fill-muted-foreground text-[10px]"
          >
            {{ tick.label }}
          </text>
        </g>

        <!-- Purchase Price Baseline (Cost Basis) -->
        <g v-if="purchaseLineY !== null">
          <line
            :x1="padding.left"
            :y1="purchaseLineY"
            :x2="svgWidth - padding.right"
            :y2="purchaseLineY"
            stroke="#6366f1"
            stroke-width="1.5"
            stroke-dasharray="4 4"
          />
          <text
            :x="svgWidth - padding.right + 4"
            :y="purchaseLineY + 3.5"
            class="fill-indigo-500 text-[10px] font-semibold"
          >
            {{ $t("components.item.price_chart.cost_basis") }}
          </text>
        </g>

        <!-- Area Gradient Fill -->
        <path :d="areaPath" :fill="isPositiveTrend ? 'url(#priceGradientPositive)' : 'url(#priceGradientNegative)'" />

        <!-- Price Line Path -->
        <path
          :d="linePath"
          fill="none"
          :stroke="isPositiveTrend ? '#10b981' : '#f43f5e'"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />

        <!-- Data Points (Hoverable) -->
        <g class="data-points">
          <circle
            v-for="(pt, idx) in points"
            :key="idx"
            :cx="pt.x"
            :cy="pt.y"
            r="4"
            class="cursor-pointer transition-all hover:scale-150"
            :fill="isPositiveTrend ? '#10b981' : '#f43f5e'"
            stroke="white"
            stroke-width="2"
            @mouseenter="hoveredPoint = pt"
          />
        </g>

        <!-- Hover Indicator Line -->
        <g v-if="hoveredPoint">
          <line
            :x1="hoveredPoint.x"
            :y1="padding.top"
            :x2="hoveredPoint.x"
            :y2="svgHeight - padding.bottom"
            stroke="currentColor"
            class="text-foreground/40"
            stroke-width="1"
            stroke-dasharray="2 2"
          />
          <circle
            :cx="hoveredPoint.x"
            :cy="hoveredPoint.y"
            r="6"
            :fill="isPositiveTrend ? '#10b981' : '#f43f5e'"
            stroke="white"
            stroke-width="2"
          />
        </g>
      </svg>

      <!-- Hover Tooltip Overlay -->
      <div
        v-if="hoveredPoint"
        class="pointer-events-none absolute z-10 -translate-x-1/2 rounded-md border bg-popover/95 px-2.5 py-1.5 text-xs shadow-md backdrop-blur-sm"
        :style="{
          left: `${(hoveredPoint.x / svgWidth) * 100}%`,
          top: `${Math.max(5, (hoveredPoint.y / svgHeight) * 100 - 30)}%`,
        }"
      >
        <div class="font-bold text-popover-foreground">
          {{ formatCurrency(hoveredPoint.entry.price) }}
        </div>
        <div class="text-[10px] text-muted-foreground">
          {{ new Date(hoveredPoint.entry.recordedAt).toLocaleDateString(undefined, { dateStyle: "medium" }) }}
        </div>
        <div v-if="hoveredPoint.entry.source" class="text-[9px] uppercase tracking-wide text-primary">
          {{ hoveredPoint.entry.source }}
        </div>
      </div>
    </div>
  </div>
</template>

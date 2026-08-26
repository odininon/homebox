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

  const pathData = computed(() => {
    if (points.value.length === 0) return "";
    if (points.value.length === 1) {
      const p = points.value[0];
      if (!p) return "";
      return `M ${padding.left} ${p.y} L ${padding.left + chartWidth.value} ${p.y}`;
    }

    return points.value.reduce((acc, p, i) => {
      return i === 0 ? `M ${p.x} ${p.y}` : `${acc} L ${p.x} ${p.y}`;
    }, "");
  });

  const areaData = computed(() => {
    if (points.value.length === 0) return "";
    const bottomY = padding.top + chartHeight.value;
    if (points.value.length === 1) {
      const p = points.value[0];
      if (!p) return "";
      return `M ${padding.left} ${bottomY} L ${padding.left} ${p.y} L ${padding.left + chartWidth.value} ${p.y} L ${padding.left + chartWidth.value} ${bottomY} Z`;
    }

    const first = points.value[0];
    const last = points.value[points.value.length - 1];
    if (!first || !last) return "";
    return `${pathData.value} L ${last.x} ${bottomY} L ${first.x} ${bottomY} Z`;
  });

  const baselineY = computed(() => {
    if (!props.purchasePrice || props.purchasePrice <= 0) return null;
    return getY(props.purchasePrice);
  });

  // Y-Axis Grid Lines & Labels
  const yTicks = computed(() => {
    const count = 4;
    const ticks = [];
    for (let i = 0; i <= count; i++) {
      const val = priceRange.value.min + (i / count) * (priceRange.value.max - priceRange.value.min);
      ticks.push({
        value: val,
        y: getY(val),
        label: formatCurrency(val),
      });
    }
    return ticks;
  });

  // X-Axis Date Labels
  const xTicks = computed(() => {
    if (filteredEntries.value.length === 0) return [];
    const count = Math.min(4, filteredEntries.value.length);
    const step = Math.floor(filteredEntries.value.length / count) || 1;
    const ticks = [];

    for (let i = 0; i < filteredEntries.value.length; i += step) {
      const e = filteredEntries.value[i];
      if (!e) continue;
      const d = new Date(e.recordedAt);
      ticks.push({
        x: getX(e.recordedAt),
        label: d.toLocaleDateString(undefined, { month: "short", day: "numeric" }),
      });
    }
    return ticks;
  });

  // Active Hovered Point
  const hoveredPoint = ref<{ x: number; y: number; entry: PriceHistoryEntry } | null>(null);

  const gainLoss = computed(() => {
    if (!hoveredPoint.value || !props.purchasePrice || props.purchasePrice <= 0) return null;
    const diff = hoveredPoint.value.entry.price - props.purchasePrice;
    const pct = (diff / props.purchasePrice) * 100;
    return {
      diff,
      pct,
      isPositive: diff >= 0,
    };
  });
</script>

<template>
  <div class="space-y-3">
    <!-- Header with Range Toggles -->
    <div class="flex items-center justify-between">
      <div class="text-sm font-medium text-muted-foreground">
        {{ $t("items.price_history_chart", "Market Price History") }}
      </div>
      <div class="flex gap-1 rounded-lg border bg-muted/40 p-1 text-xs">
        <button
          v-for="range in ['1M', '3M', '6M', '1Y', 'ALL'] as const"
          :key="range"
          class="rounded px-2.5 py-1 font-medium transition-colors"
          :class="
            selectedRange === range
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          "
          @click="selectedRange = range"
        >
          {{ range }}
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="filteredEntries.length === 0"
      class="flex h-44 items-center justify-center rounded-lg border border-dashed p-6 text-sm text-muted-foreground"
    >
      {{
        $t("items.no_price_history", "No price history recorded yet. Click 'Sync Price' to fetch latest market data.")
      }}
    </div>

    <!-- Chart Container -->
    <div v-else class="relative overflow-hidden rounded-lg border bg-card p-2 shadow-sm">
      <!-- Tooltip Box -->
      <div
        v-if="hoveredPoint"
        class="pointer-events-none absolute z-10 rounded-lg border bg-popover/95 px-3 py-2 text-xs shadow-lg backdrop-blur-sm transition-all"
        :style="{
          left: `${Math.min(Math.max(hoveredPoint.x - 70, 10), svgWidth - 160)}px`,
          top: '10px',
        }"
      >
        <div class="font-semibold text-foreground">
          {{ formatCurrency(hoveredPoint.entry.price) }}
        </div>
        <div class="text-[11px] text-muted-foreground">
          {{
            new Date(hoveredPoint.entry.recordedAt).toLocaleDateString(undefined, {
              year: "numeric",
              month: "short",
              day: "numeric",
            })
          }}
        </div>
        <div
          v-if="hoveredPoint.entry.marketLow || hoveredPoint.entry.marketHigh"
          class="text-[10px] text-muted-foreground/80"
        >
          Range: {{ formatCurrency(hoveredPoint.entry.marketLow || 0) }} -
          {{ formatCurrency(hoveredPoint.entry.marketHigh || 0) }}
        </div>
        <div
          v-if="gainLoss"
          class="mt-1 font-medium"
          :class="gainLoss.isPositive ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'"
        >
          {{ gainLoss.isPositive ? "+" : "" }}{{ formatCurrency(gainLoss.diff) }} ({{ gainLoss.isPositive ? "+" : ""
          }}{{ gainLoss.pct.toFixed(1) }}%) vs cost
        </div>
      </div>

      <!-- SVG Chart -->
      <svg
        :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
        class="h-56 w-full touch-none select-none"
        @mouseleave="hoveredPoint = null"
      >
        <defs>
          <linearGradient id="priceGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="currentColor" class="text-primary" stop-opacity="0.35" />
            <stop offset="100%" stop-color="currentColor" class="text-primary" stop-opacity="0.0" />
          </linearGradient>
        </defs>

        <!-- Horizontal Grid Lines & Y-Labels -->
        <g class="text-muted-foreground/30">
          <line
            v-for="tick in yTicks"
            :key="tick.value"
            :x1="padding.left"
            :y1="tick.y"
            :x2="svgWidth - padding.right"
            :y2="tick.y"
            stroke="currentColor"
            stroke-dasharray="3,3"
            stroke-width="1"
          />
        </g>

        <!-- Y-Axis Labels -->
        <g class="fill-muted-foreground text-[10px]">
          <text
            v-for="tick in yTicks"
            :key="'label-' + tick.value"
            :x="padding.left - 8"
            :y="tick.y + 3"
            text-anchor="end"
          >
            {{ tick.label }}
          </text>
        </g>

        <!-- Purchase Price Baseline -->
        <g v-if="baselineY !== null">
          <line
            :x1="padding.left"
            :y1="baselineY"
            :x2="svgWidth - padding.right"
            :y2="baselineY"
            stroke="rgba(234, 88, 12, 0.85)"
            stroke-width="1.5"
            stroke-dasharray="4,4"
          />
          <text
            :x="svgWidth - padding.right - 4"
            :y="baselineY - 4"
            text-anchor="end"
            class="fill-orange-600 text-[9px] font-semibold dark:fill-orange-400"
          >
            Cost Basis: {{ formatCurrency(purchasePrice || 0) }}
          </text>
        </g>

        <!-- Area Fill -->
        <path :d="areaData" fill="url(#priceGradient)" />

        <!-- Line Path -->
        <path
          :d="pathData"
          fill="none"
          stroke="currentColor"
          class="text-primary"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />

        <!-- X-Axis Labels -->
        <g class="fill-muted-foreground text-[10px]">
          <text v-for="(tick, idx) in xTicks" :key="idx" :x="tick.x" :y="svgHeight - 10" text-anchor="middle">
            {{ tick.label }}
          </text>
        </g>

        <!-- Interactive Points -->
        <g>
          <circle
            v-for="(p, idx) in points"
            :key="idx"
            :cx="p.x"
            :cy="p.y"
            r="4.5"
            class="cursor-pointer fill-background stroke-primary stroke-[2.5] transition-all hover:stroke-[3.5]"
            @mouseenter="hoveredPoint = p"
          />
        </g>
      </svg>
    </div>
  </div>
</template>

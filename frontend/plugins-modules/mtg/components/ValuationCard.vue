<script setup lang="ts">
  import { computed, ref, onMounted } from "vue";
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import type { EntityOut, PriceHistoryEntry } from "~~/lib/api/types/data-contracts";
  import { useFormatCurrency } from "~/composables/use-formatters";
  import BaseCard from "@/components/Base/Card.vue";
  import { Button } from "@/components/ui/button";
  import { DialogRoot } from "reka-ui";
  import { DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import PriceHistoryChart from "./PriceHistoryChart.vue";
  import MdiRefresh from "~icons/mdi/refresh";
  import MdiTrendingUp from "~icons/mdi/trending-up";
  import MdiPlus from "~icons/mdi/plus";
  import MdiDelete from "~icons/mdi/delete";
  import MdiAutoFix from "~icons/mdi/auto-fix";

  const props = defineProps<{
    item: EntityOut;
  }>();

  const emit = defineEmits<{
    (e: "refresh"): void;
  }>();

  const { t } = useI18n();
  const api = useUserApi();

  const fmtCurrency = ref<((v: number | string) => string) | null>(null);
  const formatCurrency = (val: number | string) => {
    if (fmtCurrency.value) {
      return fmtCurrency.value(val);
    }
    return `$${Number(val || 0).toFixed(2)}`;
  };

  const priceHistory = ref<PriceHistoryEntry[]>([]);
  const loadingHistory = ref(false);
  const syncing = ref(false);
  const autoDetecting = ref(false);

  // Manual snapshot dialog state
  const manualDialogOpen = ref(false);
  const manualPrice = ref<number>();
  const manualDate = ref<string>(new Date().toISOString().split("T")[0] || "");
  const manualNotes = ref<string>("");

  const showSnapshotTable = ref(false);

  // Check if item has custom fields with TCGPlayer link
  const detectedTCGLink = computed(() => {
    if (!props.item.fields) return null;
    for (const f of props.item.fields) {
      if (f.textValue && /tcgplayer\.com\/(?:product\/|magic\/product\/show\?id=)(\d+)/i.test(f.textValue)) {
        return {
          fieldName: f.name,
          url: f.textValue,
        };
      }
    }
    return null;
  });

  const latestPrice = computed<number>(() => {
    if (priceHistory.value.length > 0) {
      const sorted = [...priceHistory.value].sort(
        (a, b) => new Date(b.recordedAt).getTime() - new Date(a.recordedAt).getTime()
      );
      return sorted[0]?.price ?? (props.item.currentMarketPrice || 0);
    }
    return props.item.currentMarketPrice || 0;
  });

  const totalMarketValue = computed(() => {
    const qty = props.item.quantity || 1;
    return latestPrice.value * qty;
  });

  const totalCostBasis = computed(() => {
    const qty = props.item.quantity || 1;
    return (props.item.purchasePrice || 0) * qty;
  });

  const gainLoss = computed(() => {
    const cost = totalCostBasis.value;
    const market = totalMarketValue.value;
    if (cost <= 0 && market <= 0) return null;

    const diff = market - cost;
    const pct = cost > 0 ? (diff / cost) * 100 : 0;
    return {
      diff,
      pct,
      isPositive: diff >= 0,
    };
  });

  async function loadPriceHistory() {
    if (!props.item.id) return;
    loadingHistory.value = true;
    try {
      const { data, error } = await api.items.pricing.getPrices(props.item.id);
      if (error) {
        toast.error(t("items.toast.failed_load_prices"));
        return;
      }
      priceHistory.value = data || [];
    } finally {
      loadingHistory.value = false;
    }
  }

  async function syncPrice() {
    if (!props.item.id) return;
    syncing.value = true;
    try {
      const { data, error } = await api.items.pricing.syncPrice(props.item.id);
      if (error) {
        toast.error(t("items.toast.failed_sync_price"));
        return;
      }
      toast.success(
        t("items.toast.price_synced", {
          price: formatCurrency(data.price),
        })
      );
      await loadPriceHistory();
      emit("refresh");
    } finally {
      syncing.value = false;
    }
  }

  async function autoDetectAndSync() {
    if (!props.item.id) return;
    autoDetecting.value = true;
    try {
      const { error } = await api.items.pricing.autoDetectPricing(props.item.id);
      if (error) {
        toast.error(t("items.toast.failed_detect_tcg"));
        return;
      }
      toast.success(t("items.toast.tcg_detected"));
      await loadPriceHistory();
      emit("refresh");
    } finally {
      autoDetecting.value = false;
    }
  }

  async function createManualSnapshot() {
    if (!props.item.id || !manualPrice.value) return;
    try {
      const { error } = await api.items.pricing.create(props.item.id, {
        price: Number(manualPrice.value),
        recordedAt: (manualDate.value ? new Date(manualDate.value) : new Date()).toISOString(),
        notes: manualNotes.value || "",
        source: "manual",
        sourceId: "",
        marketMid: Number(manualPrice.value),
        marketLow: 0,
        marketHigh: 0,
        directLow: 0,
      });
      if (error) {
        toast.error(t("items.toast.failed_create_price"));
        return;
      }
      toast.success(t("items.toast.price_created"));
      manualDialogOpen.value = false;
      manualPrice.value = undefined;
      manualNotes.value = "";
      await loadPriceHistory();
      emit("refresh");
    } catch {
      toast.error(t("items.toast.failed_create_price"));
    }
  }

  async function deleteSnapshot(priceId: string) {
    if (!props.item.id) return;
    try {
      const { error } = await api.items.pricing.delete(props.item.id, priceId);
      if (error) {
        toast.error(t("items.toast.failed_delete_price"));
        return;
      }
      toast.success(t("items.toast.price_deleted"));
      await loadPriceHistory();
      emit("refresh");
    } catch {
      toast.error(t("items.toast.failed_delete_price"));
    }
  }

  onMounted(async () => {
    fmtCurrency.value = await useFormatCurrency();
    await loadPriceHistory();
  });
</script>

<template>
  <BaseCard>
    <!-- Header with Live Price and Sync Actions -->
    <div class="flex flex-wrap items-center justify-between gap-3 border-b pb-4">
      <div class="flex items-center gap-2.5">
        <div class="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <MdiTrendingUp class="size-5" />
        </div>
        <div>
          <h3 class="text-base font-semibold text-foreground">
            {{ $t("components.item.valuation.title") }}
          </h3>
          <p class="text-xs text-muted-foreground">
            <span v-if="item.priceTrackingEnabled && item.priceTrackingSource">
              {{ $t("components.item.valuation.tracked_via", { source: item.priceTrackingSource.toUpperCase() }) }}
              <span v-if="item.priceTrackingId" class="text-foreground/70">(ID: {{ item.priceTrackingId }})</span>
            </span>
            <span v-else-if="detectedTCGLink">
              {{ $t("components.item.valuation.detected_hint", { field: detectedTCGLink.fieldName }) }}
            </span>
            <span v-else>
              {{ $t("components.item.valuation.untracked_hint") }}
            </span>
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-2">
        <!-- Auto Detect & Sync Button -->
        <Button
          v-if="!item.priceTrackingEnabled && detectedTCGLink"
          size="sm"
          variant="secondary"
          class="h-8 gap-1.5 text-xs font-medium"
          :disabled="autoDetecting"
          @click="autoDetectAndSync"
        >
          <MdiAutoFix class="size-3.5 text-primary" :class="{ 'animate-spin': autoDetecting }" />
          <span>{{ autoDetecting ? $t("global.detecting") : $t("items.enable_and_sync") }}</span>
        </Button>

        <!-- Live Sync Button -->
        <Button
          v-if="item.priceTrackingEnabled"
          size="sm"
          variant="outline"
          class="h-8 gap-1.5 text-xs font-medium"
          :disabled="syncing"
          @click="syncPrice"
        >
          <MdiRefresh class="size-3.5" :class="{ 'animate-spin': syncing }" />
          <span>{{ syncing ? $t("global.syncing") : $t("items.sync_price") }}</span>
        </Button>

        <!-- Manual Snapshot Button -->
        <Button size="sm" variant="outline" class="h-8 gap-1.5 text-xs font-medium" @click="manualDialogOpen = true">
          <MdiPlus class="size-3.5" />
          <span>{{ $t("items.add_price_snapshot") }}</span>
        </Button>
      </div>
    </div>

    <!-- Summary Valuation Metric Cards Grid -->
    <div class="my-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <!-- Latest Market Unit Price -->
      <div class="flex flex-col rounded-lg border bg-muted/20 p-3">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("items.market_unit_price") }}</span>
        <span class="mt-1 text-lg font-bold text-foreground sm:text-xl">
          {{ latestPrice > 0 ? formatCurrency(latestPrice) : "--" }}
        </span>
        <span v-if="item.lastPriceSyncAt" class="text-[10px] text-muted-foreground">
          {{ $t("items.synced_at", { date: new Date(item.lastPriceSyncAt).toLocaleDateString() }) }}
        </span>
      </div>

      <!-- Total Market Value -->
      <div class="flex flex-col rounded-lg border bg-muted/20 p-3">
        <span class="text-xs font-medium text-muted-foreground">{{
          $t("items.total_market_value", { qty: item.quantity || 1 })
        }}</span>
        <span class="mt-1 text-lg font-bold text-foreground sm:text-xl">
          {{ totalMarketValue > 0 ? formatCurrency(totalMarketValue) : "--" }}
        </span>
      </div>

      <!-- Total Cost Basis -->
      <div class="flex flex-col rounded-lg border bg-muted/20 p-3">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("items.cost_basis") }}</span>
        <span class="mt-1 text-lg font-bold text-muted-foreground sm:text-xl">
          {{ totalCostBasis > 0 ? formatCurrency(totalCostBasis) : "--" }}
        </span>
      </div>

      <!-- Unrealized Gain / Loss -->
      <div class="flex flex-col rounded-lg border bg-muted/20 p-3">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("items.unrealized_return") }}</span>
        <div v-if="gainLoss" class="mt-1 flex items-center gap-1.5">
          <span
            class="text-lg font-bold sm:text-xl"
            :class="gainLoss.isPositive ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'"
          >
            {{ gainLoss.isPositive ? "+" : "" }}{{ formatCurrency(gainLoss.diff) }}
          </span>
          <span
            class="inline-flex items-center rounded-full px-1.5 py-0.5 text-[11px] font-semibold"
            :class="
              gainLoss.isPositive
                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                : 'bg-rose-500/10 text-rose-600 dark:text-rose-400'
            "
          >
            {{ gainLoss.isPositive ? "▲" : "▼" }} {{ Math.abs(gainLoss.pct).toFixed(1) }}%
          </span>
        </div>
        <span v-else class="mt-1 text-lg font-bold text-muted-foreground sm:text-xl">--</span>
      </div>
    </div>

    <!-- Interactive Price History SVG Chart -->
    <div class="mt-4 border-t pt-4">
      <PriceHistoryChart
        :entries="priceHistory"
        :purchase-price="item.purchasePrice"
        :purchase-date="item.purchaseDate"
      />
    </div>

    <!-- Collapsible Snapshots Table -->
    <div v-if="priceHistory.length > 0" class="mt-4 border-t pt-3">
      <button
        type="button"
        class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground"
        @click="showSnapshotTable = !showSnapshotTable"
      >
        <span>{{
          showSnapshotTable
            ? $t("items.hide_history_table")
            : $t("items.show_history_table", { count: priceHistory.length })
        }}</span>
      </button>

      <div v-if="showSnapshotTable" class="mt-3 max-h-60 overflow-y-auto rounded-lg border">
        <table class="w-full text-left text-xs">
          <thead class="bg-muted/50 text-muted-foreground">
            <tr>
              <th class="p-2 font-medium">{{ $t("global.date") }}</th>
              <th class="p-2 font-medium">{{ $t("global.price") }}</th>
              <th class="p-2 font-medium">{{ $t("global.source") }}</th>
              <th class="p-2 font-medium">{{ $t("global.notes") }}</th>
              <th class="p-2 text-right font-medium">{{ $t("global.actions") }}</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="entry in priceHistory" :key="entry.id" class="hover:bg-muted/30">
              <td class="whitespace-nowrap p-2 text-muted-foreground">
                {{ new Date(entry.recordedAt).toLocaleString() }}
              </td>
              <td class="p-2 font-bold text-foreground">
                {{ formatCurrency(entry.price) }}
              </td>
              <td class="p-2 uppercase text-muted-foreground">
                {{ entry.source }}
              </td>
              <td class="max-w-xs truncate p-2 text-muted-foreground">
                {{ entry.notes || "--" }}
              </td>
              <td class="p-2 text-right">
                <Button
                  size="icon"
                  variant="ghost"
                  class="size-6 text-muted-foreground hover:text-destructive"
                  @click="deleteSnapshot(entry.id)"
                >
                  <MdiDelete class="size-3.5" />
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Manual Price Snapshot Dialog -->
    <DialogRoot v-model:open="manualDialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t("items.add_manual_snapshot_title") }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-3">
          <div class="space-y-1.5">
            <Label for="manualPrice">{{ $t("items.price_value") }}</Label>
            <Input id="manualPrice" v-model.number="manualPrice" type="number" step="any" placeholder="149.99" />
          </div>
          <div class="space-y-1.5">
            <Label for="manualDate">{{ $t("items.snapshot_date") }}</Label>
            <Input id="manualDate" v-model="manualDate" type="date" />
          </div>
          <div class="space-y-1.5">
            <Label for="manualNotes">{{ $t("items.snapshot_notes") }}</Label>
            <Input id="manualNotes" v-model="manualNotes" placeholder="e.g. eBay sold listing, local card shop price" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="manualDialogOpen = false">{{ $t("global.cancel") }}</Button>
          <Button :disabled="!manualPrice || manualPrice <= 0" @click="createManualSnapshot">{{
            $t("global.save")
          }}</Button>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  </BaseCard>
</template>

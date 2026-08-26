<script setup lang="ts">
  import { computed, ref, onMounted } from "vue";
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import type { EntityOut, PriceHistoryEntry } from "~~/lib/api/types/data-contracts";
  import { useFormatCurrency } from "~/composables/use-formatters";
  import BaseCard from "@/components/Base/Card.vue";
  import { Button } from "@/components/ui/button";
  import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import PriceHistoryChart from "./PriceHistoryChart.vue";
  import MdiRefresh from "~icons/mdi/refresh";
  import MdiTrendingUp from "~icons/mdi/trending-up";
  import MdiTrendingDown from "~icons/mdi/trending-down";
  import MdiPlus from "~icons/mdi/plus";
  import MdiDelete from "~icons/mdi/delete";
  import MdiAutoFix from "~icons/mdi/auto-fix";
  import MdiOpenInNew from "~icons/mdi/open-in-new";

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
  const manualPrice = ref<number | null>(null);
  const manualDate = ref<string>(new Date().toISOString().split("T")[0]);
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
      return sorted[0].price;
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
    if (!props.item?.id) return;
    loadingHistory.value = true;
    try {
      const { data, error } = await api.items.pricing.getPrices(props.item.id);
      if (!error && data) {
        priceHistory.value = data;
      }
    } catch {
      // Ignored
    } finally {
      loadingHistory.value = false;
    }
  }

  async function syncPrice() {
    if (!props.item?.id) return;
    syncing.value = true;
    try {
      const { data, error } = await api.items.pricing.syncPrice(props.item.id);
      if (error) {
        toast.error(t("items.toast.sync_price_failed", "Failed to sync market price. Check if TCGPlayer product ID is valid."));
        return;
      }
      if (data) {
        toast.success(t("items.toast.sync_price_success", `Market price updated to ${formatCurrency(data.price)}`));
        await loadPriceHistory();
        emit("refresh");
      }
    } catch {
      toast.error(t("items.toast.sync_price_failed", "Failed to sync market price."));
    } finally {
      syncing.value = false;
    }
  }

  async function autoDetectLink() {
    if (!props.item?.id) return;
    autoDetecting.value = true;
    try {
      const { data, error } = await api.items.pricing.autoDetectPricing(props.item.id);
      if (error) {
        toast.error(t("items.toast.autodetect_failed", "Could not auto-detect TCGPlayer link from custom fields."));
        return;
      }
      if (data) {
        toast.success(t("items.toast.autodetect_success", "TCGPlayer link detected and price tracking enabled!"));
        await loadPriceHistory();
        emit("refresh");
      }
    } catch {
      toast.error(t("items.toast.autodetect_failed", "Failed to auto-detect price tracking link."));
    } finally {
      autoDetecting.value = false;
    }
  }

  async function addManualSnapshot() {
    if (!props.item?.id || manualPrice.value === null || manualPrice.value < 0) {
      toast.error(t("items.toast.invalid_price", "Please enter a valid price."));
      return;
    }

    const { data, error } = await api.items.pricing.create(props.item.id, {
      price: manualPrice.value,
      recordedAt: new Date(manualDate.value || Date.now()),
      notes: manualNotes.value,
      source: "manual",
      marketLow: 0,
      marketMid: 0,
      marketHigh: 0,
      directLow: 0,
      sourceId: "",
    });

    if (error) {
      toast.error(t("items.toast.add_price_failed", "Failed to add price snapshot."));
      return;
    }

    toast.success(t("items.toast.add_price_success", "Price snapshot recorded!"));
    manualDialogOpen.value = false;
    manualPrice.value = null;
    manualNotes.value = "";
    await loadPriceHistory();
    emit("refresh");
  }

  async function deleteSnapshot(priceId: string) {
    if (!props.item?.id) return;
    const { error } = await api.items.pricing.delete(props.item.id, priceId);
    if (error) {
      toast.error(t("items.toast.delete_price_failed", "Failed to delete price snapshot."));
      return;
    }
    toast.success(t("items.toast.delete_price_success", "Price snapshot deleted."));
    await loadPriceHistory();
    emit("refresh");
  }

  onMounted(async () => {
    fmtCurrency.value = await useFormatCurrency();
    loadPriceHistory();
  });
</script>

<template>
  <BaseCard collapsable>
    <template #title>
      <div class="flex items-center gap-2">
        <MdiTrendingUp class="size-5 text-emerald-600 dark:text-emerald-400" />
        <span>{{ $t("items.market_valuation", "Market Valuation & Price History") }}</span>
      </div>
    </template>

    <template #title-actions>
      <div class="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          class="h-8 gap-1.5 text-xs font-medium"
          :disabled="syncing"
          @click="syncPrice"
        >
          <MdiRefresh class="size-3.5" :class="{ 'animate-spin': syncing }" />
          <span>{{ syncing ? $t("items.syncing", "Syncing...") : $t("items.sync_price", "Sync Price") }}</span>
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="h-8 gap-1 text-xs"
          @click="manualDialogOpen = true"
        >
          <MdiPlus class="size-3.5" />
          <span>{{ $t("items.add_price", "Add Snapshot") }}</span>
        </Button>
      </div>
    </template>

    <div class="space-y-5">
      <!-- Auto-detect banner if custom field has TCG link but tracking disabled -->
      <div
        v-if="detectedTCGLink && !item.priceTrackingEnabled"
        class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-primary/30 bg-primary/5 p-3 text-xs"
      >
        <div class="flex items-center gap-2">
          <MdiAutoFix class="size-4 text-primary" />
          <span>
            Detected TCGPlayer link in custom field <strong>"{{ detectedTCGLink.fieldName }}"</strong>
          </span>
        </div>
        <Button
          size="sm"
          variant="default"
          class="h-7 gap-1 text-xs"
          :disabled="autoDetecting"
          @click="autoDetectLink"
        >
          <MdiRefresh v-if="autoDetecting" class="size-3 animate-spin" />
          <span>{{ autoDetecting ? 'Enabling...' : 'Enable Automated Sync' }}</span>
        </Button>
      </div>

      <!-- Top Metric Cards Grid -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <!-- Current Market Price -->
        <div class="rounded-lg border bg-muted/20 p-3">
          <div class="text-xs font-medium text-muted-foreground">
            {{ $t("items.current_market_price", "Market Price") }} (unit)
          </div>
          <div class="mt-1 text-lg font-bold text-foreground">
            {{ formatCurrency(latestPrice) }}
          </div>
          <div v-if="item.priceTrackingSource" class="mt-0.5 text-[10px] text-muted-foreground">
            Source: <span class="capitalize font-medium">{{ item.priceTrackingSource }}</span>
          </div>
        </div>

        <!-- Total Market Value -->
        <div class="rounded-lg border bg-muted/20 p-3">
          <div class="text-xs font-medium text-muted-foreground">
            {{ $t("items.total_market_value", "Total Value") }} (qty: {{ item.quantity || 1 }})
          </div>
          <div class="mt-1 text-lg font-bold text-foreground">
            {{ formatCurrency(totalMarketValue) }}
          </div>
          <div v-if="item.lastPriceSyncAt" class="mt-0.5 text-[10px] text-muted-foreground">
            Synced: {{ new Date(item.lastPriceSyncAt).toLocaleDateString() }}
          </div>
        </div>

        <!-- Cost Basis -->
        <div class="rounded-lg border bg-muted/20 p-3">
          <div class="text-xs font-medium text-muted-foreground">
            {{ $t("items.cost_basis", "Cost Basis") }}
          </div>
          <div class="mt-1 text-lg font-bold text-foreground">
            {{ formatCurrency(totalCostBasis) }}
          </div>
          <div v-if="item.purchasePrice" class="mt-0.5 text-[10px] text-muted-foreground">
            Unit Cost: {{ formatCurrency(item.purchasePrice) }}
          </div>
          <div v-else class="mt-0.5 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
            $0 / Company Perk
          </div>
        </div>

        <!-- Unrealized Gain / Loss -->
        <div class="rounded-lg border bg-muted/20 p-3">
          <div class="text-xs font-medium text-muted-foreground">
            {{ $t("items.gain_loss", "Profit / ROI") }}
          </div>
          <div
            v-if="gainLoss"
            class="mt-1 flex items-baseline gap-1.5 text-lg font-bold"
            :class="gainLoss.isPositive ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'"
          >
            <span>{{ gainLoss.isPositive ? '+' : '' }}{{ formatCurrency(gainLoss.diff) }}</span>
            <span v-if="item.purchasePrice > 0" class="text-xs font-semibold">
              ({{ gainLoss.isPositive ? '+' : '' }}{{ gainLoss.pct.toFixed(1) }}%)
            </span>
          </div>
          <div v-else class="mt-1 text-lg font-bold text-muted-foreground">
            --
          </div>
          <div class="mt-0.5 text-[10px] text-muted-foreground">
            Unrealized ROI
          </div>
        </div>
      </div>

      <!-- Price History Line Chart -->
      <PriceHistoryChart
        :entries="priceHistory"
        :purchase-price="item.purchasePrice"
        :purchase-date="item.purchaseDate"
      />

      <!-- Collapsible Snapshot History Table -->
      <div class="pt-2">
        <button
          type="button"
          class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors"
          @click="showSnapshotTable = !showSnapshotTable"
        >
          <span>{{ showSnapshotTable ? '▼ Hide' : '▶ Show' }} Snapshot History ({{ priceHistory.length }})</span>
        </button>

        <div v-if="showSnapshotTable && priceHistory.length > 0" class="mt-3 overflow-x-auto rounded-lg border">
          <table class="w-full text-left text-xs">
            <thead class="border-b bg-muted/40 font-medium text-muted-foreground">
              <tr>
                <th class="p-2.5">Date</th>
                <th class="p-2.5">Market Price</th>
                <th class="p-2.5">Low / High</th>
                <th class="p-2.5">Source</th>
                <th class="p-2.5">Product Notes</th>
                <th class="p-2.5 text-right">Action</th>
              </tr>
            </thead>
            <tbody class="divide-y">
              <tr v-for="entry in priceHistory" :key="entry.id" class="hover:bg-muted/20">
                <td class="p-2.5 font-medium">
                  {{ new Date(entry.recordedAt).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' }) }}
                </td>
                <td class="p-2.5 font-bold text-foreground">
                  {{ formatCurrency(entry.price) }}
                </td>
                <td class="p-2.5 text-muted-foreground">
                  <span v-if="entry.marketLow || entry.marketHigh">
                    {{ formatCurrency(entry.marketLow || 0) }} - {{ formatCurrency(entry.marketHigh || 0) }}
                  </span>
                  <span v-else>--</span>
                </td>
                <td class="p-2.5">
                  <span class="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] uppercase">
                    {{ entry.source }}
                  </span>
                </td>
                <td class="p-2.5 text-muted-foreground max-w-xs truncate">
                  {{ entry.notes || '--' }}
                </td>
                <td class="p-2.5 text-right">
                  <Button
                    size="icon"
                    variant="ghost"
                    class="size-7 text-destructive hover:bg-destructive/10"
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
    </div>

    <!-- Manual Price Snapshot Dialog -->
    <Dialog v-model:open="manualDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ $t("items.add_manual_price", "Add Price Snapshot") }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-1.5">
            <Label for="manual-price">Price Amount ($)</Label>
            <Input
              id="manual-price"
              v-model.number="manualPrice"
              type="number"
              step="0.01"
              placeholder="e.g. 294.99"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="manual-date">Date</Label>
            <Input
              id="manual-date"
              v-model="manualDate"
              type="date"
            />
          </div>
          <div class="space-y-1.5">
            <Label for="manual-notes">Notes (Optional)</Label>
            <Input
              id="manual-notes"
              v-model="manualNotes"
              placeholder="e.g. TCGPlayer Market / Local Game Store / Buylist"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="manualDialogOpen = false">
            {{ $t("global.cancel", "Cancel") }}
          </Button>
          <Button @click="addManualSnapshot">
            {{ $t("global.save", "Save Snapshot") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </BaseCard>
</template>

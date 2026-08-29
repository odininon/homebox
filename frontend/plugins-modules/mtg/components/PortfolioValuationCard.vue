<template>
  <Card
    v-if="hasTrackedItems"
    class="relative overflow-hidden border border-primary/20 bg-gradient-to-br from-card to-primary/[0.03] p-5 shadow-sm"
  >
    <div class="flex flex-wrap items-center justify-between gap-4 border-b pb-4">
      <div class="flex items-center gap-3">
        <div class="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <MdiTrendingUp class="size-6" />
        </div>
        <div>
          <h3 class="text-base font-semibold text-foreground">
            {{ $t("home.portfolio_title") }}
          </h3>
          <p class="text-xs text-muted-foreground">
            {{ $t("home.portfolio_subtitle") }}
          </p>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          class="h-8 gap-1.5 text-xs font-medium"
          :disabled="syncing"
          @click="syncAllPrices"
        >
          <MdiRefresh class="size-3.5" :class="{ 'animate-spin': syncing }" />
          <span>{{ syncing ? $t("global.syncing") : $t("items.sync_all_prices") }}</span>
        </Button>
        <Button size="sm" class="h-8 gap-1.5 text-xs font-medium" @click="openDialog(DialogID.MtgSearch)">
          <MdiCardsOutline class="size-3.5" />
          <span>{{ $t("menu.search_mtg") }}</span>
        </Button>
      </div>
    </div>

    <!-- Portfolio Metrics Grid -->
    <div class="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
      <!-- Total Market Value -->
      <div class="flex flex-col space-y-1">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("home.total_market_value") }}</span>
        <span class="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
          {{ formatCurrency(marketValue) }}
        </span>
      </div>

      <!-- Total Cost Basis -->
      <div class="flex flex-col space-y-1">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("home.cost_basis") }}</span>
        <span class="text-xl font-bold tracking-tight text-muted-foreground sm:text-2xl">
          {{ formatCurrency(costBasis) }}
        </span>
      </div>

      <!-- Unrealized Gain / Loss -->
      <div class="flex flex-col space-y-1">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("home.unrealized_gain_loss") }}</span>
        <div v-if="gainLoss" class="flex items-center gap-1.5">
          <span
            class="text-xl font-bold tracking-tight sm:text-2xl"
            :class="gainLoss.isPositive ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'"
          >
            {{ gainLoss.isPositive ? "+" : "" }}{{ formatCurrency(gainLoss.diff) }}
          </span>
          <span
            class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold"
            :class="
              gainLoss.isPositive
                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                : 'bg-rose-500/10 text-rose-600 dark:text-rose-400'
            "
          >
            {{ gainLoss.isPositive ? "▲" : "▼" }} {{ Math.abs(gainLoss.pct).toFixed(1) }}%
          </span>
        </div>
        <span v-else class="text-xl font-bold tracking-tight text-muted-foreground sm:text-2xl">--</span>
      </div>

      <!-- Tracked Items Count -->
      <div class="flex flex-col space-y-1">
        <span class="text-xs font-medium text-muted-foreground">{{ $t("home.tracked_products") }}</span>
        <div class="flex items-center gap-2">
          <span class="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            {{ trackedCount }}
          </span>
          <span
            class="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary"
          >
            <span class="size-1.5 animate-pulse rounded-full bg-emerald-500" />
            <span>{{ $t("home.live_tracking") }}</span>
          </span>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
  import { computed, ref, onMounted } from "vue";
  import { useI18n } from "vue-i18n";
  import { toast } from "@/components/ui/sonner";
  import type { GroupStatistics } from "~~/lib/api/types/data-contracts";
  import { useFormatCurrency } from "~/composables/use-formatters";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  import { Card } from "@/components/ui/card";
  import { Button } from "@/components/ui/button";
  import MdiTrendingUp from "~icons/mdi/trending-up";
  import MdiRefresh from "~icons/mdi/refresh";
  import MdiCardsOutline from "~icons/mdi/cards-outline";

  const props = defineProps<{
    stats?: GroupStatistics;
  }>();

  const emit = defineEmits<{
    (e: "refresh"): void;
  }>();

  const { t } = useI18n();
  const api = useUserApi();
  const { openDialog } = useDialog();

  const syncing = ref(false);

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

  const hasTrackedItems = computed(() => {
    return (props.stats?.totalTrackedItems ?? 0) > 0;
  });

  const trackedCount = computed(() => {
    return props.stats?.totalTrackedItems ?? 0;
  });

  const marketValue = computed(() => {
    return props.stats?.totalTrackedMarketValue ?? 0;
  });

  const costBasis = computed(() => {
    return props.stats?.totalTrackedCostBasis ?? 0;
  });

  const gainLoss = computed(() => {
    if (costBasis.value <= 0 && marketValue.value <= 0) return null;
    const diff = marketValue.value - costBasis.value;
    const pct = costBasis.value > 0 ? (diff / costBasis.value) * 100 : 0;
    return {
      diff,
      pct,
      isPositive: diff >= 0,
    };
  });

  async function syncAllPrices() {
    syncing.value = true;
    try {
      const { data, error } = await api.items.pricing.syncAll();
      if (error) {
        toast.error(t("items.toast.failed_sync_all"));
        return;
      }
      toast.success(
        t("items.toast.sync_all_success", {
          count: data?.updatedCount ?? 0,
        })
      );
      emit("refresh");
    } finally {
      syncing.value = false;
    }
  }
</script>

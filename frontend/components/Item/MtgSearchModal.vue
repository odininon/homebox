<template>
  <Dialog :dialog-id="DialogID.MtgSearch">
    <DialogContent class="flex max-h-[90vh] w-full max-w-4xl flex-col p-6">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2 text-xl">
          <MdiCardsOutline class="size-6 text-primary" />
          <span>{{ $t("components.item.mtg_search.title", "Search MTG Sealed Products") }}</span>
        </DialogTitle>
      </DialogHeader>

      <!-- Search Bar & Suggestions -->
      <div class="space-y-3 pt-2">
        <div class="flex items-center gap-2">
          <div class="relative flex-1">
            <MdiMagnify class="absolute left-3 top-1/2 size-5 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref="searchInputRef"
              v-model="searchQuery"
              class="h-11 pl-10 text-base"
              :placeholder="
                $t('components.item.mtg_search.placeholder', 'Search booster boxes, bundles, commander decks, sets...')
              "
              @keyup.enter="performSearch"
            />
          </div>
          <Button class="h-11 gap-2 px-5" :disabled="searching" @click="performSearch">
            <MdiLoading v-if="searching" class="size-4 animate-spin" />
            <span v-else>{{ $t("global.search", "Search") }}</span>
          </Button>
        </div>

        <!-- Quick filter / suggestion tags -->
        <div class="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
          <span class="font-medium text-foreground/70">{{
            $t("components.item.mtg_search.quick_tags", "Popular:")
          }}</span>
          <button
            v-for="tag in popularTags"
            :key="tag"
            type="button"
            class="rounded-full border border-transparent bg-secondary/70 px-2.5 py-0.5 transition-colors hover:bg-primary/20 hover:text-primary"
            @click="quickSearch(tag)"
          >
            {{ tag }}
          </button>
        </div>
      </div>

      <Separator class="my-3" />

      <!-- Error State -->
      <div
        v-if="errorMessage"
        class="flex items-center gap-2 rounded-md border border-destructive bg-destructive/10 p-3 text-sm text-destructive"
      >
        <MdiAlertCircleOutline class="size-5 shrink-0" />
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Results Section -->
      <div class="min-h-[300px] flex-1 overflow-y-auto pr-1">
        <!-- Loading State -->
        <div v-if="searching" class="flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground">
          <MdiLoading class="size-8 animate-spin text-primary" />
          <p class="text-sm">
            {{ $t("components.item.mtg_search.searching", "Searching MTG catalog & live market prices...") }}
          </p>
        </div>

        <!-- Empty / Initial State -->
        <div
          v-else-if="!results || results.length === 0"
          class="flex flex-col items-center justify-center gap-2 py-16 text-center text-muted-foreground"
        >
          <MdiCardsOutline class="size-12 text-primary opacity-30" />
          <p class="text-base font-medium text-foreground/80">
            {{
              searchQuery
                ? $t("components.item.mtg_search.no_results", "No sealed products found")
                : $t(
                    "components.item.mtg_search.start_search",
                    "Search for MTG sealed booster boxes, cases, bundles, and decks"
                  )
            }}
          </p>
          <p class="max-w-md text-xs">
            {{
              searchQuery
                ? $t(
                    "components.item.mtg_search.no_results_tip",
                    "Try searching for set name (e.g. 'Modern Horizons 3', 'Innistrad', 'Commander Masters') or product type ('Play Booster Display', 'Bundle')."
                  )
                : $t(
                    "components.item.mtg_search.start_tip",
                    "Select any product to instantly create an inventory item with box artwork, set tagging, and automatic TCGPlayer price tracking."
                  )
            }}
          </p>
        </div>

        <!-- Results Grid -->
        <div v-else class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div
            v-for="prod in results"
            :key="prod.productId"
            class="group relative flex cursor-pointer flex-col justify-between rounded-xl border bg-card p-3.5 shadow-sm transition-all hover:border-primary/50 hover:shadow-md"
            @click="selectProduct(prod)"
          >
            <div class="flex items-start gap-3.5">
              <!-- Box Art Image -->
              <div
                class="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/30"
              >
                <img
                  v-if="prod.imageUrl"
                  :src="prod.imageUrl"
                  :alt="prod.name"
                  class="size-full object-contain transition-transform group-hover:scale-105"
                  loading="lazy"
                />
                <MdiCardsOutline v-else class="size-8 text-muted-foreground/40" />
              </div>

              <!-- Product Details -->
              <div class="min-w-0 flex-1">
                <div class="mb-1 flex items-center gap-1.5">
                  <span
                    class="inline-flex max-w-[200px] items-center truncate rounded-md bg-secondary px-2 py-0.5 text-[11px] font-medium text-secondary-foreground"
                  >
                    {{ prod.groupName }}
                  </span>
                  <span class="text-[10px] text-muted-foreground">ID: {{ prod.productId }}</span>
                </div>
                <h4
                  class="line-clamp-2 text-sm font-semibold leading-snug text-foreground transition-colors group-hover:text-primary"
                >
                  {{ prod.name }}
                </h4>
              </div>
            </div>

            <!-- Price & Action Footer -->
            <div class="mt-3 flex items-center justify-between border-t pt-2.5">
              <div class="flex items-baseline gap-1.5">
                <span class="text-xs text-muted-foreground">{{ $t("items.market_price", "Market:") }}</span>
                <span v-if="prod.marketPrice > 0" class="text-base font-bold text-emerald-600 dark:text-emerald-400">
                  {{ formatCurrency(prod.marketPrice) }}
                </span>
                <span v-else class="text-xs italic text-muted-foreground">
                  {{ $t("items.price_untracked", "N/A") }}
                </span>
              </div>

              <div class="flex items-center gap-2">
                <a
                  v-if="prod.url"
                  :href="prod.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                  :title="$t('components.item.mtg_search.view_tcgplayer', 'View on TCGPlayer')"
                  @click.stop
                >
                  <MdiOpenInNew class="size-4" />
                </a>
                <Button size="sm" class="h-8 gap-1 text-xs font-medium" @click.stop="selectProduct(prod)">
                  <MdiPlus class="size-3.5" />
                  <span>{{ $t("global.select", "Select & Import") }}</span>
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <DialogFooter class="mt-4 flex items-center justify-between border-t pt-2 sm:justify-between">
        <span class="text-xs text-muted-foreground">
          {{
            results && results.length > 0
              ? $t("components.item.mtg_search.results_count", { count: results.length })
              : ""
          }}
        </span>
        <Button variant="outline" @click="closeDialog(DialogID.MtgSearch)">
          {{ $t("global.cancel", "Cancel") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
  import { ref, onMounted, onUnmounted, nextTick } from "vue";
  import { useI18n } from "vue-i18n";
  import { DialogID } from "@/components/ui/dialog-provider/utils";
  import { useDialog } from "~/components/ui/dialog-provider";
  import { useFormatCurrency } from "~/composables/use-formatters";
  import type { ProductSearchResult } from "~~/lib/api/types/data-contracts";
  import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
  import { Button } from "~/components/ui/button";
  import { Input } from "~/components/ui/input";
  import { Separator } from "@/components/ui/separator";
  import MdiCardsOutline from "~icons/mdi/cards-outline";
  import MdiMagnify from "~icons/mdi/magnify";
  import MdiLoading from "~icons/mdi/loading";
  import MdiPlus from "~icons/mdi/plus";
  import MdiOpenInNew from "~icons/mdi/open-in-new";
  import MdiAlertCircleOutline from "~icons/mdi/alert-circle-outline";

  const emit = defineEmits<{
    (e: "select", product: ProductSearchResult): void;
  }>();

  const { closeDialog, openDialog, registerOpenDialogCallback } = useDialog();
  const { t } = useI18n();
  const api = useUserApi();

  const searchInputRef = ref<HTMLInputElement | null>(null);
  const searchQuery = ref("");
  const searching = ref(false);
  const results = ref<ProductSearchResult[] | null>(null);
  const errorMessage = ref<string | null>(null);

  const fmtCurrency = ref<((v: number | string) => string) | null>(null);
  const formatCurrency = (val: number | string) => {
    if (fmtCurrency.value) {
      return fmtCurrency.value(val);
    }
    return `$${Number(val || 0).toFixed(2)}`;
  };

  const popularTags = [
    "Modern Horizons 3",
    "Commander Masters",
    "Bloomburrow",
    "Duskmourn",
    "Foundations",
    "Ravnica Remastered",
    "Dominaria Remastered",
    "Double Masters",
    "Play Booster",
    "Collector Booster",
  ];

  async function performSearch() {
    const q = searchQuery.value.trim();
    if (!q) {
      results.value = [];
      return;
    }

    searching.value = true;
    errorMessage.value = null;
    try {
      const { data, error } = await api.items.pricing.searchCatalog(q);
      if (error) {
        errorMessage.value = t("errors.api_failure", "Failed to search catalog: ") + error;
        results.value = [];
      } else {
        results.value = data || [];
      }
    } catch (err) {
      errorMessage.value = String(err);
      results.value = [];
    } finally {
      searching.value = false;
    }
  }

  function quickSearch(tag: string) {
    searchQuery.value = tag;
    performSearch();
  }

  function selectProduct(prod: ProductSearchResult) {
    emit("select", prod);
    closeDialog(DialogID.MtgSearch);

    // Open CreateModal with pre-populated MTG product data
    openDialog(DialogID.CreateEntity, {
      params: {
        baseType: "item",
        mtgProduct: prod,
      },
    });
  }

  onMounted(async () => {
    fmtCurrency.value = await useFormatCurrency();

    const cleanup = registerOpenDialogCallback(DialogID.MtgSearch, params => {
      errorMessage.value = null;
      if (params?.query) {
        searchQuery.value = params.query;
        performSearch();
      }
      nextTick(() => {
        searchInputRef.value?.focus();
      });
    });

    onUnmounted(cleanup);
  });
</script>

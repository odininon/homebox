<script setup lang="ts">
  /* eslint-disable vue/no-mutating-props, @typescript-eslint/no-explicit-any */
  import { toast } from "@/components/ui/sonner";
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  import { Card } from "~/components/ui/card";
  import { Button } from "@/components/ui/button";
  import FormTextField from "~/components/Form/TextField.vue";
  import FormCheckbox from "~/components/Form/Checkbox.vue";
  import MdiCardsOutline from "~icons/mdi/cards-outline";

  const props = defineProps<{
    item: any;
  }>();

  const { openDialog } = useDialog();

  function detectAndFillTCGLink() {
    if (!props.item?.fields) return;
    for (const f of props.item.fields) {
      if (f.textValue) {
        const match = f.textValue.match(/tcgplayer\.com\/(?:product\/|magic\/product\/show\?id=)(\d+)/i);
        if (match && match[1]) {
          props.item.priceTrackingEnabled = true;
          props.item.priceTrackingSource = "tcgplayer";
          props.item.priceTrackingId = match[1];
          toast.success(`Found TCGPlayer product ID ${match[1]} in custom field "${f.name}"`);
          return;
        }
      }
    }
    toast.info("No TCGPlayer product URL found in custom fields.");
  }

  function openMtgSearchModal() {
    openDialog(DialogID.MtgSearch, {
      params: {
        query: props.item?.name || "",
      },
    });
  }
</script>

<template>
  <Card v-if="item" class="overflow-visible shadow-xl">
    <div class="flex flex-wrap items-center justify-between gap-2 px-4 py-5 sm:px-6">
      <div>
        <h3 class="text-lg font-medium leading-6">
          {{ $t("items.price_tracking_section") }}
        </h3>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ $t("items.price_tracking_subtitle") }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <Button size="sm" variant="outline" type="button" class="h-7 gap-1 text-xs" @click="openMtgSearchModal">
          <MdiCardsOutline class="size-3.5 text-primary" />
          <span>{{ $t("items.search_mtg_catalog") }}</span>
        </Button>
        <Button size="sm" variant="outline" type="button" class="h-7 gap-1 text-xs" @click="detectAndFillTCGLink">
          <span>{{ $t("items.detect_from_fields") }}</span>
        </Button>
      </div>
    </div>
    <div class="border-t sm:p-0">
      <div class="grid grid-cols-1 sm:divide-y">
        <div class="border-b px-4 pb-4 pt-2 sm:px-6">
          <FormCheckbox v-model="item.priceTrackingEnabled" :label="$t('items.enable_price_tracking')" inline />
        </div>

        <div v-if="item.priceTrackingEnabled" class="border-b px-4 pb-4 pt-2 sm:px-6">
          <FormTextField v-model="item.priceTrackingSource" :label="$t('items.price_tracking_source')" inline />
        </div>

        <div v-if="item.priceTrackingEnabled" class="border-b px-4 pb-4 pt-2 sm:px-6">
          <FormTextField
            v-model="item.priceTrackingId"
            :label="$t('items.price_tracking_id')"
            placeholder="e.g. 541164 or https://www.tcgplayer.com/product/541164/..."
            inline
          />
        </div>

        <div v-if="item.priceTrackingEnabled" class="border-b px-4 pb-4 pt-2 sm:px-6">
          <FormTextField
            v-model.number="item.currentMarketPrice"
            type="number"
            step="any"
            :label="$t('items.current_market_price')"
            inline
          />
        </div>
      </div>
    </div>
  </Card>
</template>

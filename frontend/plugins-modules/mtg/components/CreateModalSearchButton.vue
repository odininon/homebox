<script setup lang="ts">
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  import { Button } from "@/components/ui/button";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import MdiCardsOutline from "~icons/mdi/cards-outline";

  defineProps<{
    loading?: boolean;
  }>();

  const { openDialog } = useDialog();
</script>

<template>
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button variant="outline" :disabled="loading" size="icon" @click="openDialog(DialogID.MtgSearch)">
          <MdiCardsOutline class="size-4 text-primary" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        <p>{{ $t("components.item.mtg_search.button_tooltip") }}</p>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
</template>

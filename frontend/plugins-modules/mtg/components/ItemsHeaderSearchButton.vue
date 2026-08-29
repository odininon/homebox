<script setup lang="ts">
  import { useDialog } from "@/components/ui/dialog-provider";
  import { DialogID } from "~/components/ui/dialog-provider/utils";
  import { Button } from "@/components/ui/button";
  import MdiCardsOutline from "~icons/mdi/cards-outline";

  const { openDialog } = useDialog();
</script>

<template>
  <Button size="sm" variant="outline" class="gap-1.5" @click="openDialog(DialogID.MtgSearch)">
    <MdiCardsOutline class="size-4 text-primary" />
    <span class="hidden sm:inline">{{ $t("menu.search_mtg") }}</span>
  </Button>
</template>

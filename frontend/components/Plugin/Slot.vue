<script setup lang="ts">
  /* eslint-disable @typescript-eslint/no-explicit-any */
  import { computed } from "vue";
  import { usePluginRegistry } from "~/lib/plugins/registry";

  const props = defineProps<{
    name: string;
    context?: Record<string, any>;
  }>();

  const registry = usePluginRegistry();
  const slotItems = computed(() => registry.getSlotRegistrations(props.name, props.context));
</script>

<template>
  <template v-for="item in slotItems" :key="item.id">
    <component :is="item.component" v-bind="context" />
  </template>
</template>

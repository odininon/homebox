import { pluginRegistry } from "~/lib/plugins/registry";
import { mtgPlugin } from "~/plugins-modules/mtg";

export default defineNuxtPlugin(() => {
  // Register builtin / compile-time frontend plugins
  pluginRegistry.register(mtgPlugin);

  return {
    provide: {
      pluginRegistry,
    },
  };
});

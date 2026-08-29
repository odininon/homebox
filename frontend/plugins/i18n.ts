/* eslint-disable @typescript-eslint/no-explicit-any */
import type { CompileError, MessageContext } from "vue-i18n";
import { createI18n } from "vue-i18n";
import { IntlMessageFormat } from "intl-messageformat";

import { pluginRegistry, deepMerge } from "../lib/plugins/registry";

export default defineNuxtPlugin(({ vueApp }) => {
  function checkDefaultLanguage() {
    let matched = null;
    const languages = Object.getOwnPropertyNames(messages());
    const matching = navigator.languages.filter(lang => languages.some(l => l.toLowerCase() === lang.toLowerCase()));
    if (matching.length > 0) {
      matched = matching[0];
    }
    if (!matched) {
      languages.forEach(lang => {
        const languagePartials = navigator.language.split("-")[0];
        if (lang.toLowerCase() === languagePartials) {
          matched = lang;
        }
      });
    }
    return matched;
  }
  const preferences = useViewPreferences();
  const i18n = createI18n({
    fallbackLocale: "en",
    globalInjection: true,
    legacy: false,
    locale: preferences.value.language || checkDefaultLanguage() || "en",
    messageCompiler,
    messages: messages(),
  });
  vueApp.use(i18n);

  pluginRegistry.onPluginRegistered(plugin => {
    if (plugin.messages) {
      for (const [locale, msgs] of Object.entries(plugin.messages)) {
        i18n.global.mergeLocaleMessage(locale, msgs);
      }
    }
  });

  watch(
    () => preferences.value.language,
    language => {
      if (!language) {
        return;
      }

      i18n.global.locale.value = language;
    }
  );

  return {
    provide: {
      i18nGlobal: i18n.global,
    },
  };
});

export const messages = () => {
  const messages: Record<string, any> = {};

  // 1. Core locales
  const coreModules = import.meta.glob("../locales/*.json", { eager: true });
  for (const path in coreModules) {
    const fileName = path.split("/").pop() || "";
    const key = fileName.replace(/\.json$/, "");
    if (key) {
      const content = (coreModules[path] as any)?.default ?? coreModules[path];
      messages[key] = { ...content };
    }
  }

  // 2. Plugin locales from file system glob
  const pluginModules = import.meta.glob("../plugins-modules/**/locales/*.json", { eager: true });
  for (const path in pluginModules) {
    const fileName = path.split("/").pop() || "";
    const locale = fileName.replace(/\.json$/, "");
    if (locale) {
      if (!messages[locale]) {
        messages[locale] = {};
      }
      const content = (pluginModules[path] as any)?.default ?? pluginModules[path];
      deepMerge(messages[locale], content);
    }
  }

  // 3. Plugin messages from PluginRegistry
  const registryMessages = pluginRegistry.getAllMessages();
  for (const [locale, msgs] of Object.entries(registryMessages)) {
    if (!messages[locale]) {
      messages[locale] = {};
    }
    deepMerge(messages[locale], msgs);
  }

  return messages;
};

export const messageCompiler: (
  message: string | any,
  {
    locale,
    key,
    onError,
  }: {
    locale: any;
    key: any;
    onError: any;
  }
) => (ctx: MessageContext) => unknown = (message, { locale, key, onError }) => {
  if (typeof message === "string") {
    /**
     * You can tune your message compiler performance more with your cache strategy or also memoization at here
     */
    const formatter = new IntlMessageFormat(message, locale);
    return (ctx: MessageContext) => {
      return formatter.format(ctx.values);
    };
  } else {
    /**
     * for AST.
     * If you would like to support it,
     * You need to transform locale messages such as `json`, `yaml`, etc. with the bundle plugin.
     */
    if (onError) {
      onError(new Error("not support for AST") as CompileError);
    }
    return () => key;
  }
};

/* eslint-disable @typescript-eslint/no-explicit-any */
import type { CompileError, MessageContext } from "vue-i18n";
import { IntlMessageFormat } from "intl-messageformat";

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
    const cleanLocale = typeof locale === "string" ? locale.replace(/@.*/, "") || "en" : "en";
    try {
      const formatter = new IntlMessageFormat(message, cleanLocale);
      return (ctx: MessageContext) => {
        return formatter.format(ctx.values);
      };
    } catch {
      try {
        const fallbackFormatter = new IntlMessageFormat(message, "en");
        return (ctx: MessageContext) => {
          return fallbackFormatter.format(ctx.values);
        };
      } catch {
        return () => message;
      }
    }
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

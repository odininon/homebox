/* eslint-disable @typescript-eslint/no-explicit-any */
import { describe, expect, test } from "vitest";
import { PluginRegistry, deepMerge } from "../registry";
import en from "../../../plugins-modules/mtg/locales/en.json";
import type { FrontendPlugin } from "../types";

describe("Plugin i18n and MTG translations", () => {
  test("deepMerge correctly merges nested objects without overwriting existing keys", () => {
    const target: Record<string, any> = {
      global: {
        add: "Add",
        cancel: "Cancel",
      },
      items: {
        name: "Name",
      },
    };

    const source: Record<string, any> = {
      global: {
        syncing: "Syncing...",
        date: "Date",
      },
      items: {
        market_unit_price: "Market Unit Price",
      },
      components: {
        item: {
          valuation: {
            title: "Magic & Sealed Valuation",
          },
        },
      },
    };

    const result = deepMerge(target, source);

    expect(result.global.add).toBe("Add");
    expect(result.global.cancel).toBe("Cancel");
    expect(result.global.syncing).toBe("Syncing...");
    expect(result.global.date).toBe("Date");
    expect(result.items.name).toBe("Name");
    expect(result.items.market_unit_price).toBe("Market Unit Price");
    expect(result.components.item.valuation.title).toBe("Magic & Sealed Valuation");
  });

  test("PluginRegistry registers plugin messages and triggers callback", () => {
    const registry = new PluginRegistry();
    let callbackCalled = false;

    registry.onPluginRegistered(plugin => {
      if (plugin.id === "test-plugin") {
        callbackCalled = true;
      }
    });

    const testPlugin: FrontendPlugin = {
      id: "test-plugin",
      name: "Test Plugin",
      messages: {
        en: {
          test: {
            hello: "World",
          },
        },
      },
    };

    registry.register(testPlugin);

    expect(callbackCalled).toBe(true);
    const messages = registry.getAllMessages();
    expect(messages.en?.test?.hello).toBe("World");
  });

  test("MTG plugin provides complete English translation keys", () => {
    expect(en).toBeDefined();

    expect(en?.components?.item?.valuation?.title).toBe("Magic & Sealed Valuation");
    expect(en?.components?.item?.mtg_search?.title).toBe("Search MTG Sealed Products");
    expect(en?.components?.item?.mtg_search?.select_and_import).toBe("Select & Import");
    expect(en?.components?.item?.price_chart?.title).toBe("Price History Trend");
    expect(en?.items?.market_unit_price).toBe("Market Unit Price");
    expect(en?.items?.price_tracking_section).toBe("Price Tracking & Valuation");
    expect(en?.home?.portfolio_title).toBe("MTG Sealed Portfolio & Valuation");
    expect(en?.menu?.search_mtg).toBe("MTG Sealed");
    expect(en?.global?.syncing).toBe("Syncing...");
    expect(en?.global?.detecting).toBe("Detecting...");
    expect(en?.global?.date).toBe("Date");
    expect(en?.global?.price).toBe("Price");
    expect(en?.global?.source).toBe("Source");
    expect(en?.global?.notes).toBe("Notes");
    expect(en?.global?.actions).toBe("Actions");
  });

  test("messageCompiler safely handles non-standard locale tags like en@pirate", async () => {
    const { messageCompiler } = await import("../../i18n/compiler");

    const compiler = messageCompiler("Hello {name}!", {
      locale: "en@pirate",
      key: "greeting",
      onError: () => {},
    });

    const result = (compiler as (ctx: { values: Record<string, any> }) => string)({
      values: { name: "Ahoy" },
    });

    expect(result).toBe("Hello Ahoy!");
  });
});

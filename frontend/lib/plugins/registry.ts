/* eslint-disable @typescript-eslint/no-explicit-any */
import type { Component } from "vue";
import type { FrontendPlugin, PluginComponentRegistration, PluginHeaderAction } from "./types";

export function deepMerge<T extends Record<string, any>>(target: T, source: Record<string, any>): T {
  for (const key of Object.keys(source)) {
    const srcVal = source[key];
    const tgtVal = (target as Record<string, any>)[key];
    if (
      srcVal &&
      typeof srcVal === "object" &&
      !Array.isArray(srcVal) &&
      tgtVal &&
      typeof tgtVal === "object" &&
      !Array.isArray(tgtVal)
    ) {
      deepMerge(tgtVal, srcVal);
    } else {
      (target as Record<string, any>)[key] = srcVal;
    }
  }
  return target;
}

export class PluginRegistry {
  private plugins = new Map<string, FrontendPlugin>();
  private slotComponents = new Map<string, PluginComponentRegistration[]>();
  private dialogs = new Map<string, Component>();
  private headerActions = new Map<string, PluginHeaderAction[]>();
  private onRegisterCallbacks: ((plugin: FrontendPlugin) => void)[] = [];

  public register(plugin: FrontendPlugin): void {
    if (this.plugins.has(plugin.id)) {
      return;
    }

    this.plugins.set(plugin.id, plugin);

    // Register slots
    if (plugin.slots) {
      for (const [slotName, entries] of Object.entries(plugin.slots)) {
        if (!this.slotComponents.has(slotName)) {
          this.slotComponents.set(slotName, []);
        }
        const existing = this.slotComponents.get(slotName)!;

        for (let i = 0; i < entries.length; i++) {
          const entry = entries[i];
          if (typeof entry === "object" && entry !== null && "component" in entry) {
            existing.push(entry as PluginComponentRegistration);
          } else {
            existing.push({
              id: `${plugin.id}-${slotName}-${i}`,
              component: entry as Component,
              priority: 10,
            });
          }
        }
      }
    }

    // Register dialogs
    if (plugin.dialogs) {
      for (const [dialogId, component] of Object.entries(plugin.dialogs)) {
        this.dialogs.set(dialogId, component);
      }
    }

    // Register header actions
    if (plugin.headerActions) {
      for (const [location, actions] of Object.entries(plugin.headerActions)) {
        if (!this.headerActions.has(location)) {
          this.headerActions.set(location, []);
        }
        this.headerActions.get(location)!.push(...actions);
      }
    }

    if (plugin.init) {
      plugin.init();
    }

    for (const callback of this.onRegisterCallbacks) {
      try {
        callback(plugin);
      } catch (err) {
        console.error(`Error in plugin register callback for ${plugin.id}:`, err);
      }
    }
  }

  public onPluginRegistered(callback: (plugin: FrontendPlugin) => void): () => void {
    this.onRegisterCallbacks.push(callback);
    return () => {
      this.onRegisterCallbacks = this.onRegisterCallbacks.filter(cb => cb !== callback);
    };
  }

  public getAllMessages(): Record<string, Record<string, any>> {
    const allMessages: Record<string, Record<string, any>> = {};
    for (const plugin of this.plugins.values()) {
      if (plugin.messages) {
        for (const [locale, msgs] of Object.entries(plugin.messages)) {
          if (!allMessages[locale]) {
            allMessages[locale] = {};
          }
          deepMerge(allMessages[locale], msgs);
        }
      }
    }
    return allMessages;
  }

  public getSlotRegistrations(slotName: string, context?: any): PluginComponentRegistration[] {
    const list = this.slotComponents.get(slotName) || [];
    return list
      .filter(item => {
        if (!item.when) return true;
        try {
          return item.when(context);
        } catch {
          return false;
        }
      })
      .sort((a, b) => (b.priority ?? 10) - (a.priority ?? 10));
  }

  public getDialog(dialogId: string): Component | undefined {
    return this.dialogs.get(dialogId);
  }

  public getAllDialogs(): { id: string; component: Component }[] {
    const res: { id: string; component: Component }[] = [];
    for (const [id, component] of this.dialogs.entries()) {
      res.push({ id, component });
    }
    return res;
  }

  public getHeaderActions(location: string, context?: any): PluginHeaderAction[] {
    const list = this.headerActions.get(location) || [];
    return list
      .filter(item => {
        if (!item.when) return true;
        try {
          return item.when(context);
        } catch {
          return false;
        }
      })
      .sort((a, b) => (b.priority ?? 10) - (a.priority ?? 10));
  }

  public getPlugins(): FrontendPlugin[] {
    return Array.from(this.plugins.values());
  }
}

export const pluginRegistry = new PluginRegistry();

export function usePluginRegistry(): PluginRegistry {
  return pluginRegistry;
}

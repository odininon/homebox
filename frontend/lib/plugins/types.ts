/* eslint-disable @typescript-eslint/no-explicit-any */
import type { Component } from "vue";

export interface PluginComponentRegistration {
  id: string;
  component: Component;
  priority?: number;
  when?: (context: any) => boolean;
}

export interface PluginHeaderAction {
  id: string;
  label: string;
  icon?: Component;
  onClick: (context: any) => void;
  priority?: number;
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  when?: (context: any) => boolean;
}

export interface FrontendPlugin {
  id: string;
  name: string;
  description?: string;
  init?: () => void;
  slots?: Record<string, (PluginComponentRegistration | Component)[]>;
  dialogs?: Record<string, Component>;
  headerActions?: Record<string, PluginHeaderAction[]>;
  messages?: Record<string, Record<string, any>>;
}

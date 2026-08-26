import type { Column, ColumnDef } from "@tanstack/vue-table";
import { h } from "vue";
import DropdownAction from "./data-table-dropdown.vue";
import { ArrowDown, ArrowUpDown, Check, X } from "lucide-vue-next";
import Button from "~/components/ui/button/Button.vue";
import Checkbox from "~/components/Form/Checkbox.vue";
import type { EntitySummary } from "~/lib/api/types/data-contracts";

import Currency from "~/components/global/Currency.vue";
import DateTime from "~/components/global/DateTime.vue";
import { cn } from "~/lib/utils";

/**
 * Create columns with i18n support.
 * Pass `t` from useI18n() when creating the columns in your component.
 */
export function makeColumns({
  t,
  refresh,
  disableSort,
}: {
  t: (key: string) => string;
  refresh?: () => void;
  disableSort?: boolean;
}): ColumnDef<EntitySummary>[] {
  const sortable = (column: Column<EntitySummary, unknown>, key: string, fallback?: string) => {
    const label = t(key) === key && fallback ? fallback : t(key);
    const sortState = column.getIsSorted(); // 'asc' | 'desc' | false
    if (!sortState) {
      // show the neutral up/down icon when not sorted
      return [label, h(ArrowUpDown, { class: cn(["ml-2 h-4 w-4 opacity-40", disableSort && "opacity-0"]) })];
    }
    // show a single arrow that points up for asc (rotate-180) and down for desc
    return [
      label,
      h(ArrowDown, {
        class: cn([
          "ml-2 h-4 w-4 transition-transform opacity-100",
          sortState === "asc" ? "rotate-180" : "",
          disableSort && "opacity-0",
        ]),
      }),
    ];
  };

  return [
    {
      id: "select",
      header: ({ table }) =>
        h(Checkbox, {
          modelValue: table.getIsAllPageRowsSelected()
            ? true
            : table.getSelectedRowModel().rows.length > 0
              ? ("indeterminate" as unknown as boolean) // :)
              : false,
          "onUpdate:modelValue": (value: boolean) => table.toggleAllPageRowsSelected(!!value),
          ariaLabel: t("components.item.view.selectable.select_all"),
        }),
      cell: ({ row }) =>
        h(Checkbox, {
          modelValue: row.getIsSelected(),
          "onUpdate:modelValue": (value: boolean) => row.toggleSelected(!!value),
          ariaLabel: t("components.item.view.selectable.select_row"),
        }),
      enableHiding: false,
    },
    {
      id: "assetId",
      accessorKey: "assetId",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.asset_id")
        ),
      cell: ({ row }) => h("div", { class: "text-sm" }, String(row.getValue("assetId") ?? "")),
    },
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.name")
        ),
      cell: ({ row }) => h("span", { class: "text-sm font-medium" }, row.getValue("name")),
    },
    {
      id: "quantity",
      accessorKey: "quantity",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.quantity")
        ),
      cell: ({ row }) => h("div", { class: "text-center" }, String(row.getValue("quantity") ?? "")),
    },
    {
      id: "insured",
      accessorKey: "insured",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.insured")
        ),
      cell: ({ row }) => {
        const val = row.getValue("insured");
        return h(
          "div",
          { class: "block mx-auto w-min" },
          val ? h(Check, { class: "h-4 w-4 text-green-500" }) : h(X, { class: "h-4 w-4 text-destructive" })
        );
      },
    },
    {
      id: "purchasePrice",
      accessorKey: "purchasePrice",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.purchase_price")
        ),
      cell: ({ row }) =>
        h("div", { class: "text-center" }, h(Currency, { amount: Number(row.getValue("purchasePrice")) })),
    },
    {
      id: "currentMarketPrice",
      accessorKey: "currentMarketPrice",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.market_price", "Market Price")
        ),
      cell: ({ row }) => {
        const item = row.original as EntitySummary;
        const val = Number(row.getValue("currentMarketPrice") || 0);
        if (val <= 0 && !item.priceTrackingEnabled) {
          return h("div", { class: "text-center text-xs text-muted-foreground" }, "-");
        }
        return h(
          "div",
          { class: "text-center font-medium text-emerald-600 dark:text-emerald-400" },
          val > 0
            ? h(Currency, { amount: val })
            : h("span", { class: "text-xs italic text-muted-foreground" }, "Pending")
        );
      },
    },
    {
      id: "totalMarketValue",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.total_market_value", "Total Value")
        ),
      accessorFn: (row: EntitySummary) => {
        const unit = row.currentMarketPrice > 0 ? row.currentMarketPrice : row.purchasePrice;
        return unit * (row.quantity || 1);
      },
      cell: ({ row }) => {
        const item = row.original as EntitySummary;
        const unit = item.currentMarketPrice > 0 ? item.currentMarketPrice : item.purchasePrice;
        const total = unit * (item.quantity || 1);
        if (total <= 0) {
          return h("div", { class: "text-center text-xs text-muted-foreground" }, "-");
        }
        return h("div", { class: "text-center font-semibold" }, h(Currency, { amount: total }));
      },
    },
    {
      id: "gainLoss",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.gain_loss", "Gain / Loss")
        ),
      accessorFn: (row: EntitySummary) => {
        const cost = (row.purchasePrice || 0) * (row.quantity || 1);
        const market = (row.currentMarketPrice || 0) * (row.quantity || 1);
        if (cost <= 0 || market <= 0) return 0;
        return market - cost;
      },
      cell: ({ row }) => {
        const item = row.original as EntitySummary;
        const cost = (item.purchasePrice || 0) * (item.quantity || 1);
        const market = (item.currentMarketPrice || 0) * (item.quantity || 1);
        if (cost <= 0 || market <= 0) {
          return h("div", { class: "text-center text-xs text-muted-foreground" }, "-");
        }
        const diff = market - cost;
        const pct = (diff / cost) * 100;
        const isPos = diff >= 0;
        return h(
          "div",
          {
            class: [
              "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold mx-auto",
              isPos
                ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                : "bg-rose-500/10 text-rose-600 dark:text-rose-400",
            ].join(" "),
          },
          [
            isPos ? "+" : "",
            h(Currency, { amount: diff }),
            ` (${isPos ? "+" : ""}${pct.toFixed(0)}%)`,
          ]
        );
      },
    },
    {
      id: "location",
      accessorKey: "location",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.location")
        ),
      cell: ({ row }) => {
        const item = row.original as EntitySummary;
        const loc = item.parent as { id: string; name: string } | null;
        if (loc) {
          return h("a", { href: `/location/${loc.id}`, class: "hover:underline text-sm" }, loc.name);
        }
        return h("div", { class: "text-sm text-muted-foreground" }, "");
      },
    },
    {
      id: "archived",
      accessorKey: "archived",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.archived")
        ),
      cell: ({ row }) => {
        const val = row.getValue("archived");
        return h(
          "div",
          { class: "block mx-auto w-min" },
          val ? h(Check, { class: "h-4 w-4 text-green-500" }) : h(X, { class: "h-4 w-4 text-destructive" })
        );
      },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.created_at")
        ),
      cell: ({ row }) =>
        h(
          "div",
          { class: "text-center text-sm" },
          h(DateTime, { date: row.getValue("createdAt") as Date, datetimeType: "date" })
        ),
    },
    {
      id: "updatedAt",
      accessorKey: "updatedAt",
      header: ({ column }) =>
        h(
          Button,
          {
            variant: "ghost",
            onClick: () => !disableSort && column.toggleSorting(column.getIsSorted() === "asc"),
          },
          () => sortable(column, "items.updated_at")
        ),
      cell: ({ row }) =>
        h(
          "div",
          { class: "text-center text-sm" },
          h(DateTime, { date: row.getValue("updatedAt") as Date, datetimeType: "date" })
        ),
    },
    {
      id: "actions",
      enableHiding: false,
      header: ({ table }) => {
        const selectedCount = table.getSelectedRowModel().rows.length;
        return h(
          "div",
          {
            class: [
              "relative inline-flex items-center",
              selectedCount === 0 ? "opacity-50 pointer-events-none" : "",
            ].join(" "),
          },
          [
            h(DropdownAction, {
              multi: {
                items: table.getSelectedRowModel().rows,
                columns: table.getAllColumns(),
              },
              onExpand: () => {
                table.getSelectedRowModel().rows.forEach(row => row.toggleExpanded());
              },
              view: "table",
              onRefresh: () => refresh?.(),
              table,
            }),
            selectedCount > 0 &&
              h(
                "span",
                {
                  class: "-right-1 -top-1 absolute flex size-4",
                },
                h(
                  "span",
                  {
                    class:
                      "relative flex size-4 items-center justify-center rounded-full bg-primary p-1 text-primary-foreground text-xs pointer-events-none whitespace-nowrap",
                  },
                  String(selectedCount)
                )
              ),
          ]
        );
      },
      cell: ({ row, table }) => {
        const item = row.original;
        return h(
          "div",
          { class: "relative" },
          h(DropdownAction, {
            item,
            onExpand: row.toggleExpanded,
            view: "table",
            onRefresh: () => refresh?.(),
            table,
          })
        );
      },
    },
  ];
}

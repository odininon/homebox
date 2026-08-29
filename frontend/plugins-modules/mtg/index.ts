import type { FrontendPlugin } from "~/lib/plugins/types";
import en from "./locales/en.json";
import MtgSearchModal from "./components/MtgSearchModal.vue";
import PortfolioValuationCard from "./components/PortfolioValuationCard.vue";
import ValuationCard from "./components/ValuationCard.vue";
import ItemEditPricingSection from "./components/ItemEditPricingSection.vue";
import ItemsHeaderSearchButton from "./components/ItemsHeaderSearchButton.vue";
import CreateModalSearchButton from "./components/CreateModalSearchButton.vue";

export const mtgPlugin: FrontendPlugin = {
  id: "mtg",
  name: "Magic: The Gathering Sealed Product Tracker",
  description: "Live market pricing, sealed product search, and historical valuation powered by TCGPlayer.",
  messages: {
    en,
  },
  slots: {
    "global:dialogs": [
      {
        id: "mtg-search-dialog",
        component: MtgSearchModal,
        priority: 10,
      },
    ],
    "dashboard:top": [
      {
        id: "mtg-portfolio-valuation",
        component: PortfolioValuationCard,
        priority: 20,
      },
    ],
    "item:details:valuation": [
      {
        id: "mtg-item-valuation-card",
        component: ValuationCard,
        priority: 20,
      },
    ],
    "item:edit:pricing": [
      {
        id: "mtg-item-edit-pricing",
        component: ItemEditPricingSection,
        priority: 10,
      },
    ],
    "items:list:actions": [
      {
        id: "mtg-items-header-search-btn",
        component: ItemsHeaderSearchButton,
        priority: 15,
      },
    ],
    "entity:create:actions": [
      {
        id: "mtg-create-modal-search-btn",
        component: CreateModalSearchButton,
        priority: 15,
      },
    ],
  },
  dialogs: {
    MtgSearch: MtgSearchModal,
  },
};

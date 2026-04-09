import { createRouter, createWebHashHistory, type LocationQuery } from "vue-router"
import { resolveDesignLabAccess } from "@/lib/design-lab-access"

const designLabAccess = resolveDesignLabAccess(import.meta.env.DEV)

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      component: () => import("@/layouts/AppShell.vue"),
      children: [
        {
          path: "",
          redirect: { name: "library" },
        },
        {
          path: "library",
          name: "library",
          component: () => import("@/views/LibraryView.vue"),
        },
        {
          path: "favorites",
          name: "favorites",
          component: () => import("@/views/LibraryView.vue"),
        },
        {
          path: "recent",
          name: "recent",
          redirect: (to) => {
            const query: LocationQuery = { ...to.query }
            if (query.from === "recent") {
              delete query.from
            }
            return { name: "library", query }
          },
        },
        {
          path: "tags",
          name: "tags",
          component: () => import("@/views/LibraryView.vue"),
        },
        {
          path: "trash",
          name: "trash",
          component: () => import("@/views/LibraryView.vue"),
        },
        {
          path: "actors",
          name: "actors",
          component: () => import("@/views/ActorsView.vue"),
        },
        {
          path: "history",
          name: "history",
          component: () => import("@/views/HistoryView.vue"),
        },
        {
          path: "curated-frames",
          name: "curated-frames",
          component: () => import("@/views/CuratedFramesView.vue"),
        },
        {
          path: "detail/:id",
          name: "detail",
          component: () => import("@/views/DetailView.vue"),
        },
        {
          path: "player/:id?",
          name: "player",
          component: () => import("@/views/PlayerView.vue"),
        },
        {
          path: "settings",
          name: "settings",
          component: () => import("@/views/SettingsView.vue"),
        },
        {
          path: "design-lab",
          name: "design-lab",
          beforeEnter: () => (designLabAccess.enabled ? true : designLabAccess.fallbackTarget!),
          component: () => import("@/views/DesignLabView.vue"),
        },
        {
          path: ":pathMatch(.*)*",
          name: "not-found",
          component: () => import("@/views/NotFoundView.vue"),
        },
      ],
    },
  ],
})

export default router

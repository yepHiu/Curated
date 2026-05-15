import { createRouter, createWebHashHistory, type LocationQuery } from "vue-router"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/lock",
      name: "lock",
      component: () => import("@/views/LockView.vue"),
    },
    {
      path: "/",
      component: () => import("@/layouts/AppShell.vue"),
      children: [
        {
          path: "",
          name: "home",
          component: () => import("@/views/HomeView.vue"),
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
          path: ":pathMatch(.*)*",
          name: "not-found",
          component: () => import("@/views/NotFoundView.vue"),
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (!isAuthLockEnabled() || to.name === "lock") {
    return true
  }
  try {
    const status = await authLockService.refreshStatus()
    if (status.pinEnabled && !status.unlocked) {
      return {
        name: "lock",
        query: {
          redirect: to.fullPath,
        },
      }
    }
  } catch (error) {
    console.warn("[router] auth status check failed", error)
  }
  return true
})

export default router

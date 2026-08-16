import { createRouter, createWebHashHistory, type LocationQuery, type RouteLocationNormalized } from "vue-router"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"
import { useLibraryService } from "@/services/library-service"

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
          redirect: (to) => {
            const query: LocationQuery = { ...to.query }
            if (query.from === "tags") {
              delete query.from
            }
            return { name: "library", query }
          },
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
          path: "actors/:actorName",
          name: "actor-detail",
          component: () => import("@/views/ActorDetailView.vue"),
        },
        {
          path: "history",
          name: "history",
          component: () => import("@/views/HistoryView.vue"),
        },
        {
          path: "insights",
          name: "insights",
          component: () => import("@/views/InsightsView.vue"),
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
    prefetchPlaybackForRoute(to)
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
  prefetchPlaybackForRoute(to)
  return true
})

/**
 * Warm the playback descriptor while the player route is still resolving. For
 * HLS-eligible movies the descriptor GET also boots the server-side session, so
 * ffmpeg startup overlaps the route transition and movie hydration instead of
 * serializing after them. Best effort only; the player page fetches the
 * descriptor itself when nothing warm is available.
 */
function prefetchPlaybackForRoute(to: RouteLocationNormalized): void {
  if (to.name !== "player") return
  const id = typeof to.params.id === "string" ? to.params.id.trim() : ""
  if (!id) return
  try {
    useLibraryService().prefetchMoviePlayback(id)
  } catch {
    // Prefetch is best effort and must never block navigation.
  }
}

export default router

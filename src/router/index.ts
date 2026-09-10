import { createRouter, createWebHashHistory, type LocationQuery } from "vue-router"
import { authLockService, isAuthLockEnabled } from "@/services/auth-lock-service"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"

const comicRouteNames = new Set(["comics", "comic-detail", "comic-reader"])
const photoRouteNames = new Set(["photos", "photo-detail", "photo-viewer"])

function isComicRoute(name: unknown): boolean {
  return typeof name === "string" && comicRouteNames.has(name)
}

function isPhotoRoute(name: unknown): boolean {
  return typeof name === "string" && photoRouteNames.has(name)
}

async function guardComicRouteIfNeeded(name: unknown) {
  if (!isComicRoute(name)) {
    return true
  }
  const comicService = useComicLibraryService()
  try {
    await comicService.refreshSettings()
  } catch (error) {
    console.warn("[router] comic settings refresh failed", error)
  }
  if (!comicService.comicLibraryEnabled.value) {
    return {
      name: "settings",
      query: {
        section: "comics",
      },
    }
  }
  return true
}

async function guardPhotoRouteIfNeeded(name: unknown) {
  if (!isPhotoRoute(name)) {
    return true
  }
  const photoService = usePhotoLibraryService()
  try {
    await photoService.refreshSettings()
  } catch (error) {
    console.warn("[router] photo settings refresh failed", error)
  }
  if (!photoService.photoLibraryEnabled.value) {
    return {
      name: "settings",
      query: {
        section: "photos",
      },
    }
  }
  return true
}

async function guardOptionalMediaRouteIfNeeded(name: unknown) {
  const comicGuard = await guardComicRouteIfNeeded(name)
  if (comicGuard !== true) {
    return comicGuard
  }
  return await guardPhotoRouteIfNeeded(name)
}

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
          path: "comics",
          name: "comics",
          component: () => import("@/views/ComicsView.vue"),
        },
        {
          path: "comics/:id",
          name: "comic-detail",
          component: () => import("@/views/ComicDetailView.vue"),
        },
        {
          path: "comics/:id/read/:pageIndex?",
          name: "comic-reader",
          component: () => import("@/views/ComicReaderView.vue"),
        },
        {
          path: "photos",
          name: "photos",
          component: () => import("@/views/PhotosView.vue"),
        },
        {
          path: "photos/:id",
          name: "photo-detail",
          component: () => import("@/views/PhotoDetailView.vue"),
        },
        {
          path: "photos/:id/view/:pageIndex?",
          name: "photo-viewer",
          component: () => import("@/views/PhotoViewerView.vue"),
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
    return await guardOptionalMediaRouteIfNeeded(to.name)
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
  return await guardOptionalMediaRouteIfNeeded(to.name)
})

export default router

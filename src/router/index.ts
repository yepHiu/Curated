import { createRouter, createWebHashHistory } from "vue-router"

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
          component: () => import("@/views/LibraryView.vue"),
        },
        {
          path: "tags",
          name: "tags",
          component: () => import("@/views/LibraryView.vue"),
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

export default router

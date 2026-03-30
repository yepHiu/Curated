import { createApp } from "vue"
import "vue-sonner/style.css"
import "vue-virtual-scroller/dist/vue-virtual-scroller.css"
import App from "./App.vue"
import { i18n } from "@/i18n"
import { initClientLogger } from "@/lib/app-logger"
import { hydratePlaybackProgress } from "@/lib/playback-progress-storage"
import { hydratePlayedMovies } from "@/lib/played-movies-storage"
import router from "./router"
import "./style.css"

initClientLogger()

async function boot() {
  if (import.meta.env.VITE_USE_WEB_API === "true") {
    await Promise.all([hydratePlaybackProgress(), hydratePlayedMovies()])
  }
  createApp(App).use(i18n).use(router).mount("#app")
}

void boot()

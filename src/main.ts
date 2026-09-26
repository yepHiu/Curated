import { createApp } from "vue"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import "@fontsource-variable/outfit/wght.css"
import "vue-sonner/style.css"
import "vue-virtual-scroller/dist/vue-virtual-scroller.css"
import App from "./App.vue"
import { ensureLocaleMessages, i18n, initialLocale } from "@/i18n"
import { initClientLogger } from "@/lib/app-logger"
import { startAuthIdleLockMonitor } from "@/services/auth-idle-lock-service"
import { startProtectedWebStateBootstrap } from "@/services/protected-web-state-bootstrap"
import router from "./router"
import "./style.css"

initClientLogger()

if (window.javLibrary?.windowChrome === "macos") {
  document.documentElement.dataset.windowChrome = "macos"
}

async function boot() {
  await ensureLocaleMessages(initialLocale)
  const app = createApp(App)
  app.config.errorHandler = (err, _instance, info) => {
    console.error("[global error handler]", err, info)
  }
  app.use(i18n).use(router).mount("#app")
  startAuthIdleLockMonitor(router)
  startProtectedWebStateBootstrap()
}

void boot()

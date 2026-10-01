import type { Router } from "vue-router"
import { i18n } from "@/i18n"
import { dismissAppToast, pushAppToast } from "@/composables/use-app-toast"

const toastId = "route-load-failed"

/** A hash-only navigation does not fetch a fresh entry document on its own. */
export function reloadRouteDocument(href: string, location: Pick<Location, "href" | "replace" | "reload"> = window.location) {
  location.replace(new URL(href, location.href).href)
  location.reload()
}

/** Failed lazy imports abort navigation before Vue's render error boundary runs. */
export function installRouteLoadRecovery(router: Router) {
  router.onError((_error, to) => {
    const href = router.resolve(to.fullPath).href
    pushAppToast(i18n.global.t("app.routeLoadFailed"), {
      id: toastId,
      variant: "destructive",
      durationMs: Number.POSITIVE_INFINITY,
      action: {
        label: i18n.global.t("app.reload"),
        onClick: () => reloadRouteDocument(href),
      },
    })
  })
  router.afterEach((_to, _from, failure) => {
    if (!failure) dismissAppToast(toastId)
  })
}

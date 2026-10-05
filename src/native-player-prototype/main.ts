import { createApp } from "vue"
import "@fontsource-variable/noto-sans/wght.css"
import "@fontsource-variable/noto-sans-jp/wght.css"
import "@fontsource-variable/outfit/wght.css"
import "./style.css"
import { applyThemeToDocument, getStoredThemePreference } from "@/lib/theme-storage"
import { htmlLangFor, resolveInitialLocale } from "@/lib/locale-storage"
import NativePlayerPrototype from "./NativePlayerPrototype.vue"

document.documentElement.lang = htmlLangFor(resolveInitialLocale())
applyThemeToDocument(getStoredThemePreference())
createApp(NativePlayerPrototype).mount("#app")

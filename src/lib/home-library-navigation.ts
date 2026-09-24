import type { InjectionKey } from "vue"

export const openLibraryFromHomeKey: InjectionKey<() => void> = Symbol("openLibraryFromHome")

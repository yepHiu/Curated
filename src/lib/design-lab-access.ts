export type DesignLabFallbackTarget = {
  name: "settings"
  query: {
    section: "about"
  }
}

export type DesignLabAccess = {
  enabled: boolean
  fallbackTarget: DesignLabFallbackTarget | null
}

export function resolveDesignLabAccess(isDev: boolean): DesignLabAccess {
  if (isDev) {
    return {
      enabled: true,
      fallbackTarget: null,
    }
  }

  return {
    enabled: false,
    fallbackTarget: {
      name: "settings",
      query: {
        section: "about",
      },
    },
  }
}

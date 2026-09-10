import { describe, expect, it } from "vitest"
import * as navigationIntent from "@/lib/navigation-intent"
import {
  buildDetailRouteFromBrowse,
  buildPlayerRouteFromBrowseIntent,
  getNavigationBackTarget,
  resolveNavigationBackLink,
} from "@/lib/navigation-intent"

describe("navigation intent helpers", () => {
  it("builds detail routes with browse context separated from return intent", () => {
    expect(
      buildDetailRouteFromBrowse(
        "movie-1",
        {
          q: "Mina",
          selected: "old-id",
          tab: "new",
          autoplay: "1",
          t: "123",
        },
        "favorites",
      ),
    ).toEqual({
      name: "detail",
      params: { id: "movie-1" },
      query: {
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
        tab: "new",
      },
    })
  })

  it("builds detail-origin filtered browse routes with a detail return intent", () => {
    const build = (
      navigationIntent as unknown as {
        buildFilteredBrowseRouteFromDetail?: (input: {
          movieId: string
          currentQuery: Record<string, string>
          sourceMode: "library" | "favorites" | "recent" | "tags" | "trash"
          kind: "tag" | "actor" | "studio"
          value: string
        }) => unknown
      }
    ).buildFilteredBrowseRouteFromDetail

    expect(build).toBeTypeOf("function")
    if (!build) return

    expect(
      build({
        movieId: "movie-1",
        currentQuery: {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
          tab: "top-rated",
        },
        sourceMode: "favorites",
        kind: "actor",
        value: "Actor A",
      }),
    ).toEqual({
      name: "favorites",
      query: {
        actor: "Actor A",
        back: "detail",
        browse: "favorites",
        selected: "movie-1",
      },
    })
  })

  it("builds browse-launched player routes that return directly to browse", () => {
    expect(
      buildPlayerRouteFromBrowseIntent(
        "movie-1",
        {
          q: "Mina",
          selected: "old-id",
          tab: "new",
        },
        "favorites",
        "browse",
      ),
    ).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "browse",
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
        tab: "new",
      },
    })
  })

  it("builds detail-launched player routes that return to detail", () => {
    expect(
      buildPlayerRouteFromBrowseIntent(
        "movie-1",
        {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
        },
        "favorites",
        "detail",
      ),
    ).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "detail",
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
      },
    })
  })

  it("parses explicit and legacy back-target query semantics", () => {
    expect(getNavigationBackTarget({ back: "home" })).toBe("home")
    expect(getNavigationBackTarget({ back: "browse" })).toBe("browse")
    expect(getNavigationBackTarget({ from: "history" })).toBe("history")
    expect(getNavigationBackTarget({ from: "curated-frames" })).toBe("curated-frames")
    expect(getNavigationBackTarget({ from: "favorites" })).toBe("detail")
  })

  it("resolves browse-launched player back links to the browse page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: {
            back: "browse",
            browse: "favorites",
            q: "Mina",
            selected: "movie-1",
            tab: "new",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backLibrary",
      to: {
        name: "favorites",
        query: {
          q: "Mina",
          selected: "movie-1",
          tab: "new",
        },
      },
    })
  })

  it("resolves detail-launched player back links to the detail page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: {
            back: "detail",
            browse: "favorites",
            q: "Mina",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backDetail",
      to: {
        name: "detail",
        params: { id: "movie-1" },
        query: {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
        },
      },
    })
  })

  it("resolves special player sources and detail routes via the same helper", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: { back: "home" },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHome",
      to: { name: "home" },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: { back: "history" },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHistory",
      to: { name: "history" },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "detail",
          query: {
            browse: "trash",
            q: "Mina",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backLibrary",
      to: {
        name: "trash",
        query: {
          q: "Mina",
          selected: "movie-1",
        },
      },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "detail",
          query: {
            back: "home",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHome",
      to: { name: "home" },
    })
  })

  it("resolves comic detail back links to the comic library", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-detail",
        query: {},
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: { name: "comics" },
    })
  })

  it("resolves comic reader back links to their recorded source route", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/comics/comic-1",
        },
      }),
    ).toEqual({
      labelKey: "shell.backDetail",
      to: "/comics/comic-1",
    })

    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/comics?q=artist&sort=fileName",
        },
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: "/comics?q=artist&sort=fileName",
    })

    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/settings?section=comics",
        },
      }),
    ).toEqual({
      labelKey: "shell.backPrevious",
      to: "/settings?section=comics",
    })
  })

  it("falls back from direct comic reader links to the comic library", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {},
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: { name: "comics" },
    })
  })
})

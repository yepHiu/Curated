import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"

import MediaStill from "./MediaStill.vue"

describe("MediaStill", () => {
  it("keeps a previously loaded image visible when the component is remounted", async () => {
    const src = "/api/library/movies/movie-1/asset/thumb?v=test-media-still"
    const first = mount(MediaStill, {
      props: {
        src,
        alt: "Movie 1",
      },
    })

    await first.get("img").trigger("load")
    expect(first.get("img").classes()).toContain("opacity-100")
    first.unmount()

    const second = mount(MediaStill, {
      props: {
        src,
        alt: "Movie 1",
      },
    })

    expect(second.find(".animate-pulse").exists()).toBe(false)
    expect(second.get("img").classes()).toContain("opacity-100")
  })
})

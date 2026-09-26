import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"

const routeState = vi.hoisted(() => ({
  name: "actor-detail",
  params: { actorName: "Mina Kaze" },
  query: {} as Record<string, unknown>,
}))
const routerPush = vi.hoisted(() => vi.fn())
const routerReplace = vi.hoisted(() => vi.fn())
const serviceState = vi.hoisted(() => ({
  movies: [] as Movie[],
}))
const serviceMocks = vi.hoisted(() => ({
  toggleFavorite: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n != null ? `${key}:${params.n}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: routerPush,
    replace: routerReplace,
  }),
}))

vi.mock("@/services/library-service", async () => {
  const { computed } = await vi.importActual<typeof import("vue")>("vue")

  return {
    useLibraryService: () => ({
      movies: computed(() => serviceState.movies),
      toggleFavorite: serviceMocks.toggleFavorite,
    }),
  }
})

vi.mock("@/components/jav-library/ActorProfileCard.vue", () => ({
  default: {
    name: "ActorProfileCard",
    props: ["actorName", "showClearFilter"],
    emits: ["resolvedName"],
    template: `<section data-actor-profile :data-actor-name="actorName" :data-show-clear-filter="String(showClearFilter)">
      <button data-resolve-name @click="$emit('resolvedName', 'Canonical Actor')" />
    </section>`,
  },
}))

vi.mock("@/components/jav-library/ActorMergeDialog.vue", () => ({
  default: {
    name: "ActorMergeDialog",
    props: ["open", "sourceName"],
    emits: ["update:open", "merged"],
    template: `<section data-actor-merge-dialog :data-open="String(open)" :data-source-name="sourceName">
      <button data-complete-merge @click="$emit('merged', 'Merged Target')" />
    </section>`,
  },
}))

vi.mock("@/components/jav-library/VirtualMovieMasonry.vue", () => ({
  default: {
    name: "VirtualMovieMasonry",
    props: [
      "movies",
      "emptyTitle",
      "emptyDescription",
      "scrollPreserveKey",
    ],
    emits: ["openDetails", "openPlayer", "toggleFavorite"],
    template: `
      <section
        data-virtual-masonry
        :data-movie-ids="movies.map((movie) => movie.id).join(',')"
        :data-empty-title="emptyTitle"
        :data-empty-description="emptyDescription"
        :data-scroll-preserve-key="scrollPreserveKey"
      >
        <button data-open-details @click="$emit('openDetails', movies[0]?.id)" />
        <button data-open-player @click="$emit('openPlayer', movies[0]?.id)" />
        <button
          data-toggle-favorite
          @click="$emit('toggleFavorite', { movieId: movies[0]?.id, nextValue: true })"
        />
      </section>
    `,
  },
}))

function movie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Movie 1",
    code: "CODE-1",
    studio: "Studio",
    actors: ["Mina Kaze"],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4.5,
    userRating: undefined,
    summary: "Summary",
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: "D:/Library/movie-1.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    ...overrides,
  }
}

async function mountPage(actorName = "Mina Kaze") {
  const { default: ActorDetailPage } = await import("./ActorDetailPage.vue")
  return mount(ActorDetailPage, {
    props: { actorName },
  })
}

describe("ActorDetailPage", () => {
  beforeEach(() => {
    routeState.name = "actor-detail"
    routeState.params = { actorName: "Mina Kaze" }
    routeState.query = {}
    routerPush.mockReset()
    routerReplace.mockReset()
    serviceMocks.toggleFavorite.mockReset()
    serviceState.movies = [
      movie({ id: "movie-1", actors: ["Mina Kaze"] }),
      movie({ id: "movie-2", actors: ["Other Actor"] }),
      movie({ id: "movie-3", actors: ["Mina Kaze", "Other Actor"] }),
    ]
  })

  it("renders an actor-owned profile and exact actor filmography", async () => {
    const wrapper = await mountPage()

    const profile = wrapper.get("[data-actor-profile]")
    const masonry = wrapper.get("[data-virtual-masonry]")

    expect(profile.attributes("data-actor-name")).toBe("Mina Kaze")
    expect(profile.attributes("data-show-clear-filter")).toBe("false")
    expect(masonry.attributes("data-movie-ids")).toBe("movie-1,movie-3")
    expect(masonry.attributes("data-empty-title")).toBe("mediaEmpty.title")
    expect(masonry.attributes("data-empty-description")).toBe("actors.detailEmptyDesc")
    expect(masonry.attributes("data-scroll-preserve-key")).toBe("actor-detail:Mina Kaze")
    expect(wrapper.text()).toContain("actors.detailMovieSection")
    expect(wrapper.text()).toContain("actors.movieCount:2")
  })

  it("keeps actor-page context when opening detail and player routes", async () => {
    const wrapper = await mountPage()

    await wrapper.get("[data-open-details]").trigger("click")
    await flushPromises()

    expect(routerPush).toHaveBeenCalledWith({
      name: "detail",
      params: { id: "movie-1" },
      query: {
        actor: "Mina Kaze",
        back: "actor",
        selected: "movie-1",
      },
    })

    await wrapper.get("[data-open-player]").trigger("click")
    await flushPromises()

    expect(routerPush).toHaveBeenLastCalledWith({
      name: "player",
      params: { id: "movie-1" },
      query: {
        actor: "Mina Kaze",
        autoplay: "1",
        back: "actor",
        selected: "movie-1",
      },
    })
  })

  it("canonicalizes alias routes and navigates to the merge target", async () => {
    const wrapper = await mountPage()
    await wrapper.get("[data-resolve-name]").trigger("click")
    await flushPromises()
    expect(routerReplace).toHaveBeenCalledWith({
      name: "actor-detail",
      params: { actorName: "Canonical Actor" },
      query: {},
    })

    await wrapper.get("[data-complete-merge]").trigger("click")
    await flushPromises()
    expect(routerReplace).toHaveBeenLastCalledWith({
      name: "actor-detail",
      params: { actorName: "Merged Target" },
      query: {},
    })
  })
})

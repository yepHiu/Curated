<script setup lang="ts">
import DesignLabNav from "@/components/design-lab/DesignLabNav.vue"
import DesignLabTopbar from "@/components/design-lab/DesignLabTopbar.vue"
import DesignLabA11ySection from "@/components/design-lab/sections/DesignLabA11ySection.vue"
import DesignLabComponentsSection from "@/components/design-lab/sections/DesignLabComponentsSection.vue"
import DesignLabMotionSection from "@/components/design-lab/sections/DesignLabMotionSection.vue"
import DesignLabPlaygroundSection from "@/components/design-lab/sections/DesignLabPlaygroundSection.vue"
import DesignLabTokensSection from "@/components/design-lab/sections/DesignLabTokensSection.vue"
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <DesignLabTopbar />
    <div class="min-h-0 flex-1 lg:grid lg:grid-cols-[15rem_minmax(0,1fr)]">
      <DesignLabNav class="hidden lg:block" />
      <main class="min-h-0 overflow-y-auto p-4 lg:p-8">
        <div class="mx-auto flex w-full max-w-7xl flex-col gap-8 lg:gap-10">
          <DesignLabTokensSection />
          <DesignLabComponentsSection />
          <DesignLabPlaygroundSection />
          <DesignLabMotionSection />
          <DesignLabA11ySection />
        </div>
      </main>
    </div>
  </div>
</template>

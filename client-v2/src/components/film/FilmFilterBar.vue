<script setup lang="ts">
export interface FilterOption {
  value: string | number
  label: string
}

export interface FilterGroup {
  key: string
  title: string
  options: FilterOption[]
  current: string | number
}

interface Props {
  groups: FilterGroup[]
}

defineProps<Props>()

const emit = defineEmits<{
  (e: 'change', payload: { key: string; value: string | number }): void
}>()

function pick(key: string, value: string | number): void {
  emit('change', { key, value })
}
</script>

<template>
  <section
    class="gf-filter-bar bg-surface rounded-[var(--gf-radius-lg)] p-[var(--gf-space-4)] flex flex-col gap-[var(--gf-space-3)]"
  >
    <div
      v-for="group in groups"
      :key="group.key"
      class="gf-filter-row flex items-start gap-[var(--gf-space-3)]"
    >
      <div
        class="gf-filter-row__title shrink-0 text-secondary text-[var(--gf-fs-sm)] font-[var(--gf-fw-medium)] pt-[6px]"
      >
        {{ group.title }}
      </div>
      <div
        class="gf-filter-row__chips flex flex-wrap gap-[var(--gf-space-2)] flex-1"
      >
        <button
          v-for="opt in group.options"
          :key="String(opt.value)"
          class="gf-filter-chip"
          :class="
            String(opt.value) === String(group.current)
              ? 'gf-filter-chip--active'
              : ''
          "
          data-focusable="true"
          tabindex="0"
          :aria-pressed="String(opt.value) === String(group.current)"
          @click="pick(group.key, opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.gf-filter-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 var(--gf-space-3);
  border-radius: var(--gf-radius-full);
  background-color: transparent;
  color: var(--gf-text-secondary);
  font-size: var(--gf-fs-sm);
  font-weight: var(--gf-fw-medium);
  border: 1px solid transparent;
  cursor: pointer;
  transition:
    background-color var(--gf-dur-fast) var(--gf-ease-standard),
    color var(--gf-dur-fast) var(--gf-ease-standard);
}

.gf-filter-chip:hover {
  background-color: rgba(255, 255, 255, 0.06);
  color: var(--gf-text-primary);
}

.gf-filter-chip--active {
  background-image: var(--gf-brand-gradient);
  color: #fff;
  font-weight: var(--gf-fw-semibold);
  box-shadow: var(--gf-shadow-purple-glow);
}

@media (max-width: 767px) {
  .gf-filter-row {
    flex-direction: column;
    align-items: stretch;
  }
  .gf-filter-row__title {
    padding-top: 0;
  }
}
</style>

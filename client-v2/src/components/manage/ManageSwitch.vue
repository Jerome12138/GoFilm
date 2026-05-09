<script setup lang="ts">
interface Props {
  modelValue: boolean
  disabled?: boolean
}
const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function toggle(): void {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    type="button"
    role="switch"
    :aria-checked="props.modelValue"
    :disabled="props.disabled"
    class="relative inline-flex h-[24px] w-[44px] items-center rounded-full transition-colors disabled:opacity-50"
    :class="props.modelValue ? 'bg-[var(--gf-brand-purple)]' : 'bg-[var(--gf-border-default)]'"
    data-focusable="true"
    @click="toggle"
  >
    <span
      class="inline-block h-[18px] w-[18px] rounded-full bg-white transition-transform"
      :class="props.modelValue ? 'translate-x-[23px]' : 'translate-x-[3px]'"
    />
  </button>
</template>

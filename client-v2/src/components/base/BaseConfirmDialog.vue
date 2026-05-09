<script setup lang="ts">
import BaseDialog from '@/components/base/BaseDialog.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { answerConfirm, confirmState } from '@/composables/useConfirm'
</script>

<template>
  <BaseDialog
    :visible="confirmState.visible"
    :title="confirmState.title"
    :width="'420px'"
    :close-on-overlay="false"
    @update:visible="(v) => !v && answerConfirm(false)"
  >
    <p
      v-if="confirmState.desc"
      class="text-secondary text-[var(--gf-fs-md)] leading-[var(--gf-lh-relaxed)]"
    >
      {{ confirmState.desc }}
    </p>
    <template #footer>
      <BaseButton variant="ghost" @click="answerConfirm(false)">
        {{ confirmState.cancelText }}
      </BaseButton>
      <BaseButton
        :variant="confirmState.danger ? 'danger' : 'gradient'"
        @click="answerConfirm(true)"
      >
        {{ confirmState.okText }}
      </BaseButton>
    </template>
  </BaseDialog>
</template>

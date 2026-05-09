<script setup lang="ts" generic="T extends Record<string, any>">
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'

export interface Column<U> {
  key: keyof U & string
  label: string
  width?: string
  align?: 'left' | 'center' | 'right'
}

export interface ManageTableProps<U extends Record<string, any>> {
  columns: Column<U>[]
  rows: U[]
  rowKey: keyof U & string
  loading?: boolean
  empty?: string
}

const props = defineProps<ManageTableProps<T>>()
defineSlots<{
  cell(props: { row: T; col: Column<T>; value: unknown }): unknown
  actions(props: { row: T }): unknown
  toolbar(): unknown
}>()
</script>

<template>
  <section class="bg-surface rounded-card shadow-card overflow-hidden">
    <header
      v-if="$slots.toolbar"
      class="px-[var(--gf-space-5)] py-[var(--gf-space-4)] border-b border-subtle flex flex-wrap gap-[var(--gf-space-3)] items-center justify-between"
    >
      <slot name="toolbar" />
    </header>

    <div v-if="props.loading" class="p-[var(--gf-space-5)] flex flex-col gap-[var(--gf-space-3)]">
      <BaseSkeleton v-for="i in 5" :key="i" shape="rect" :height="'40px'" />
    </div>

    <div v-else-if="props.rows.length === 0" class="py-[var(--gf-space-10)]">
      <BaseEmpty :description="props.empty || '暂无数据'" />
    </div>

    <div v-else class="overflow-x-auto">
      <table class="min-w-full text-sm">
        <thead>
          <tr class="border-b border-subtle text-secondary">
            <th
              v-for="col in props.columns"
              :key="col.key"
              :style="col.width ? { width: col.width } : undefined"
              class="text-left font-[var(--gf-fw-semibold)] px-[var(--gf-space-4)] py-[var(--gf-space-3)]"
              :class="{
                'text-right': col.align === 'right',
                'text-center': col.align === 'center'
              }"
            >
              {{ col.label }}
            </th>
            <th
              v-if="$slots.actions"
              class="text-right font-[var(--gf-fw-semibold)] px-[var(--gf-space-4)] py-[var(--gf-space-3)] w-[180px]"
            >
              操作
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in props.rows"
            :key="String(row[props.rowKey])"
            class="border-b border-subtle hover:bg-elevated transition-colors"
          >
            <td
              v-for="col in props.columns"
              :key="col.key"
              class="px-[var(--gf-space-4)] py-[var(--gf-space-3)] text-primary"
              :class="{
                'text-right': col.align === 'right',
                'text-center': col.align === 'center'
              }"
            >
              <slot name="cell" :row="row" :col="col" :value="row[col.key]">
                {{ row[col.key] ?? '—' }}
              </slot>
            </td>
            <td
              v-if="$slots.actions"
              class="px-[var(--gf-space-4)] py-[var(--gf-space-3)] text-right"
            >
              <slot name="actions" :row="row" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

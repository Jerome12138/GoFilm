<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { manageApi } from '@/api'
import type { FilmClass } from '@/types/manage'
import ManageInput from '@/components/manage/ManageInput.vue'
import ManageSwitch from '@/components/manage/ManageSwitch.vue'
import ManageFormField from '@/components/manage/ManageFormField.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDialog from '@/components/base/BaseDialog.vue'
import BaseEmpty from '@/components/base/BaseEmpty.vue'
import BaseSkeleton from '@/components/base/BaseSkeleton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseTag from '@/components/base/BaseTag.vue'

const tree = ref<FilmClass[]>([])
const loading = ref(true)
const dialogOpen = ref(false)
const submitting = ref(false)
const editing = ref<FilmClass | null>(null)
const form = reactive<FilmClass>({
  id: 0,
  pid: 0,
  name: '',
  ename: '',
  show: true,
  sort: 0
})

async function load(): Promise<void> {
  loading.value = true
  try {
    tree.value = await manageApi.film.classTree()
  } finally {
    loading.value = false
  }
}

function openEdit(node: FilmClass): void {
  editing.value = node
  Object.assign(form, node)
  dialogOpen.value = true
}

async function toggleShow(node: FilmClass): Promise<void> {
  await manageApi.film.classUpdate({ ...node, show: !node.show })
  await load()
}

async function submit(): Promise<void> {
  submitting.value = true
  try {
    await manageApi.film.classUpdate({ ...form })
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function remove(node: FilmClass): Promise<void> {
  if (!confirm(`确认删除分类「${node.name}」及其子分类？`)) return
  await manageApi.film.classDel(node.id)
  await load()
}

onMounted(load)
</script>

<template>
  <section class="bg-surface rounded-card shadow-card p-[var(--gf-space-5)]">
    <header class="flex items-center justify-between mb-[var(--gf-space-4)]">
      <div>
        <h2 class="text-lg font-[var(--gf-fw-semibold)]">影视分类管理</h2>
        <p class="text-sm text-muted">维护顶级与子分类的展示状态、名称与排序</p>
      </div>
      <BaseButton variant="ghost" size="sm" @click="load">
        <BaseIcon name="refresh" size="16px" /> 刷新
      </BaseButton>
    </header>

    <div v-if="loading" class="flex flex-col gap-[var(--gf-space-3)]">
      <BaseSkeleton v-for="i in 4" :key="i" shape="rect" height="56px" />
    </div>

    <BaseEmpty v-else-if="!tree.length" description="暂无分类数据" />

    <div v-else class="flex flex-col gap-[var(--gf-space-2)]">
      <article
        v-for="parent in tree"
        :key="parent.id"
        class="bg-elevated rounded-[var(--gf-radius-md)] overflow-hidden"
      >
        <header
          class="flex items-center justify-between gap-[var(--gf-space-3)] px-[var(--gf-space-4)] py-[var(--gf-space-3)] border-b border-subtle"
        >
          <div class="flex items-center gap-[var(--gf-space-3)]">
            <BaseTag variant="purple">{{ parent.name }}</BaseTag>
            <span class="text-xs text-muted font-[var(--gf-font-mono)]">
              ID {{ parent.id }} · {{ parent.children?.length ?? 0 }} 子分类
            </span>
          </div>
          <div class="flex items-center gap-[var(--gf-space-2)]">
            <ManageSwitch :model-value="parent.show" @update:model-value="toggleShow(parent)" />
            <BaseButton variant="ghost" size="sm" @click="openEdit(parent)">编辑</BaseButton>
          </div>
        </header>
        <div
          v-if="parent.children?.length"
          class="flex flex-wrap gap-[var(--gf-space-2)] p-[var(--gf-space-3)]"
        >
          <div
            v-for="child in parent.children"
            :key="child.id"
            class="flex items-center gap-[var(--gf-space-2)] px-[var(--gf-space-3)] py-[var(--gf-space-2)] bg-surface border border-subtle rounded-[var(--gf-radius-md)]"
          >
            <span :class="{ 'opacity-50 line-through': !child.show }">
              {{ child.name }}
            </span>
            <button
              class="text-xs text-muted hover:text-primary"
              type="button"
              data-focusable="true"
              @click="toggleShow(child)"
            >
              {{ child.show ? '隐藏' : '显示' }}
            </button>
            <button
              class="text-xs text-muted hover:text-primary"
              type="button"
              data-focusable="true"
              @click="openEdit(child)"
            >
              编辑
            </button>
            <button
              class="text-xs text-[var(--gf-danger)] hover:opacity-80"
              type="button"
              data-focusable="true"
              @click="remove(child)"
            >
              删除
            </button>
          </div>
        </div>
      </article>
    </div>
  </section>

  <BaseDialog v-model:visible="dialogOpen" :title="editing ? '编辑分类' : '新增分类'">
    <div class="flex flex-col gap-[var(--gf-space-4)]">
      <ManageFormField label="名称" required>
        <ManageInput v-model="form.name" />
      </ManageFormField>
      <ManageFormField label="英文名">
        <ManageInput v-model="form.ename!" />
      </ManageFormField>
      <ManageFormField label="排序">
        <ManageInput v-model="form.sort!" type="number" />
      </ManageFormField>
      <ManageFormField label="展示">
        <ManageSwitch v-model="form.show" />
      </ManageFormField>
    </div>
    <template #footer>
      <BaseButton variant="ghost" @click="dialogOpen = false">取消</BaseButton>
      <BaseButton variant="gradient" :loading="submitting" @click="submit">保存</BaseButton>
    </template>
  </BaseDialog>
</template>

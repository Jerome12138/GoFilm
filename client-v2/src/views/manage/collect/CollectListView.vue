<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { manageApi } from '@/api'
import type { CollectSource } from '@/types/manage'
import ManageTable from '@/components/manage/ManageTable.vue'
import ManageInput from '@/components/manage/ManageInput.vue'
import ManageSwitch from '@/components/manage/ManageSwitch.vue'
import ManageFormField from '@/components/manage/ManageFormField.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDialog from '@/components/base/BaseDialog.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const rows = ref<CollectSource[]>([])
const loading = ref(true)
const dialogOpen = ref(false)
const submitting = ref(false)
const editing = ref<CollectSource | null>(null)

const form = reactive<CollectSource>({
  id: 0,
  name: '',
  url: '',
  type: 'json',
  resultModel: '',
  state: true,
  syncPictures: false
})

const columns = [
  { key: 'name' as const, label: '名称' },
  { key: 'url' as const, label: '采集 URL' },
  { key: 'type' as const, label: '类型', width: '90px' },
  { key: 'state' as const, label: '状态', width: '80px', align: 'center' as const }
]

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await manageApi.collect.list()
    rows.value = Array.isArray(resp) ? (resp as CollectSource[]) : resp.list ?? []
  } finally {
    loading.value = false
  }
}

function resetForm(): void {
  form.id = 0
  form.name = ''
  form.url = ''
  form.type = 'json'
  form.resultModel = ''
  form.state = true
  form.syncPictures = false
}

function openAdd(): void {
  editing.value = null
  resetForm()
  dialogOpen.value = true
}

function openEdit(row: CollectSource): void {
  editing.value = row
  Object.assign(form, row)
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  submitting.value = true
  try {
    if (editing.value) await manageApi.collect.update({ ...form })
    else await manageApi.collect.add({ ...form })
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function toggleState(row: CollectSource): Promise<void> {
  await manageApi.collect.change({ id: row.id, status: !row.state })
  await load()
}

async function remove(row: CollectSource): Promise<void> {
  if (!confirm(`确认删除采集源「${row.name}」？`)) return
  await manageApi.collect.remove(row.id)
  await load()
}

async function startSpider(row: CollectSource): Promise<void> {
  await manageApi.collect.startSpider({ sourceId: row.id, mode: 'all' })
}

onMounted(load)
</script>

<template>
  <ManageTable
    :columns="columns"
    :rows="rows"
    row-key="id"
    :loading="loading"
    empty="暂无采集源，点击右上角新增"
  >
    <template #toolbar>
      <h2 class="text-lg font-[var(--gf-fw-semibold)]">采集源管理</h2>
      <div class="flex gap-[var(--gf-space-2)]">
        <BaseButton variant="ghost" size="sm" @click="load">
          <BaseIcon name="refresh" size="16px" /> 刷新
        </BaseButton>
        <BaseButton variant="gradient" size="sm" @click="openAdd">
          <BaseIcon name="plus" size="16px" /> 新增
        </BaseButton>
      </div>
    </template>

    <template #cell="{ row, col }">
      <BaseTag v-if="col.key === 'state'" :variant="row.state ? 'success' : 'default'">
        {{ row.state ? '启用' : '停用' }}
      </BaseTag>
      <BaseTag v-else-if="col.key === 'type'" variant="purple" size="xs">
        {{ row.type }}
      </BaseTag>
      <span v-else-if="col.key === 'url'" class="text-muted text-xs font-[var(--gf-font-mono)] break-all">
        {{ row.url }}
      </span>
      <span v-else>{{ row[col.key] ?? '—' }}</span>
    </template>

    <template #actions="{ row }">
      <div class="flex gap-[var(--gf-space-1)] justify-end">
        <BaseButton variant="ghost" size="sm" @click="startSpider(row)">采集</BaseButton>
        <BaseButton variant="ghost" size="sm" @click="toggleState(row)">
          {{ row.state ? '停用' : '启用' }}
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="openEdit(row)">编辑</BaseButton>
        <BaseButton variant="danger" size="sm" @click="remove(row)">删除</BaseButton>
      </div>
    </template>
  </ManageTable>

  <BaseDialog
    v-model:visible="dialogOpen"
    :title="editing ? '编辑采集源' : '新增采集源'"
  >
    <div class="flex flex-col gap-[var(--gf-space-4)]">
      <ManageFormField label="名称" required>
        <ManageInput v-model="form.name" placeholder="例如：飞速影视" />
      </ManageFormField>
      <ManageFormField label="采集 URL" required>
        <ManageInput v-model="form.url" placeholder="https://..." />
      </ManageFormField>
      <ManageFormField label="类型" required>
        <select
          v-model="form.type"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option value="json">JSON</option>
          <option value="xml">XML</option>
        </select>
      </ManageFormField>
      <ManageFormField label="同步图片">
        <ManageSwitch v-model="form.syncPictures!" />
      </ManageFormField>
      <ManageFormField label="启用">
        <ManageSwitch v-model="form.state" />
      </ManageFormField>
    </div>
    <template #footer>
      <BaseButton variant="ghost" @click="dialogOpen = false">取消</BaseButton>
      <BaseButton variant="gradient" :loading="submitting" @click="submit">
        保存
      </BaseButton>
    </template>
  </BaseDialog>
</template>

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
  id: '',
  name: '',
  uri: '',
  resultModel: 0,
  grade: 0,
  syncPictures: false,
  collectType: 0,
  state: true,
  interval: 0
})

const columns = [
  { key: 'name' as const, label: '名称' },
  { key: 'uri' as const, label: '采集 URI' },
  { key: 'resultModel' as const, label: '类型', width: '90px' },
  { key: 'grade' as const, label: '等级', width: '80px' },
  { key: 'state' as const, label: '状态', width: '80px', align: 'center' as const }
]

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await manageApi.collect.list()
    rows.value = Array.isArray(resp) ? resp : []
  } finally {
    loading.value = false
  }
}

function resetForm(): void {
  form.id = ''
  form.name = ''
  form.uri = ''
  form.resultModel = 0
  form.grade = 0
  form.syncPictures = false
  form.collectType = 0
  form.state = true
  form.interval = 0
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
  await manageApi.collect.change({ ...row, state: !row.state })
  await load()
}

async function remove(row: CollectSource): Promise<void> {
  if (!confirm(`确认删除采集源「${row.name}」？`)) return
  await manageApi.collect.remove(row.id)
  await load()
}

async function startSpider(row: CollectSource): Promise<void> {
  await manageApi.collect.startSpider({
    id: row.id,
    ids: [],
    time: 24,
    batch: false
  })
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
      <BaseTag v-else-if="col.key === 'resultModel'" variant="purple" size="xs">
        {{ row.resultModel === 0 ? 'JSON' : 'XML' }}
      </BaseTag>
      <BaseTag v-else-if="col.key === 'grade'" variant="default" size="xs">
        {{ row.grade === 0 ? '主站' : '附属' }}
      </BaseTag>
      <span
        v-else-if="col.key === 'uri'"
        class="text-muted text-xs font-[var(--gf-font-mono)] break-all"
      >
        {{ row.uri }}
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
      <ManageFormField label="采集 URI" required>
        <ManageInput v-model="form.uri" placeholder="https://..." />
      </ManageFormField>
      <ManageFormField label="返回类型" required>
        <select
          v-model.number="form.resultModel"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option :value="0">JSON</option>
          <option :value="1">XML</option>
        </select>
      </ManageFormField>
      <ManageFormField label="站点等级">
        <select
          v-model.number="form.grade"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option :value="0">主站</option>
          <option :value="1">附属</option>
        </select>
      </ManageFormField>
      <ManageFormField label="资源类型">
        <select
          v-model.number="form.collectType"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option :value="0">视频</option>
          <option :value="1">文章</option>
          <option :value="2">演员</option>
          <option :value="3">角色</option>
          <option :value="4">网站</option>
        </select>
      </ManageFormField>
      <ManageFormField label="采集间隔（毫秒）">
        <ManageInput v-model="form.interval" type="number" placeholder="0" />
      </ManageFormField>
      <ManageFormField label="同步图片">
        <ManageSwitch v-model="form.syncPictures" />
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

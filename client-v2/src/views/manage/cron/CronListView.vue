<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { manageApi } from '@/api'
import type { CronTask } from '@/types/manage'
import ManageTable from '@/components/manage/ManageTable.vue'
import ManageInput from '@/components/manage/ManageInput.vue'
import ManageSwitch from '@/components/manage/ManageSwitch.vue'
import ManageFormField from '@/components/manage/ManageFormField.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseDialog from '@/components/base/BaseDialog.vue'
import BaseTag from '@/components/base/BaseTag.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const rows = ref<CronTask[]>([])
const loading = ref(true)
const dialogOpen = ref(false)
const submitting = ref(false)
const editing = ref<CronTask | null>(null)
const form = reactive<CronTask>({
  id: 0,
  name: '',
  cron: '',
  jobType: 'collect',
  state: true,
  remark: ''
})

const columns = [
  { key: 'name' as const, label: '任务名' },
  { key: 'cron' as const, label: 'Cron 表达式', width: '200px' },
  { key: 'jobType' as const, label: '类型', width: '110px' },
  { key: 'state' as const, label: '状态', width: '80px', align: 'center' as const },
  { key: 'remark' as const, label: '备注' }
]

async function load(): Promise<void> {
  loading.value = true
  try {
    const resp = await manageApi.cron.list()
    rows.value = (Array.isArray(resp) ? resp : (resp as { list?: CronTask[] }).list) ?? []
  } finally {
    loading.value = false
  }
}

function openAdd(): void {
  editing.value = null
  Object.assign(form, { id: 0, name: '', cron: '', jobType: 'collect', state: true, remark: '' })
  dialogOpen.value = true
}

function openEdit(row: CronTask): void {
  editing.value = row
  Object.assign(form, row)
  dialogOpen.value = true
}

async function submit(): Promise<void> {
  submitting.value = true
  try {
    if (editing.value) await manageApi.cron.update({ ...form })
    else await manageApi.cron.add({ ...form })
    dialogOpen.value = false
    await load()
  } finally {
    submitting.value = false
  }
}

async function toggleState(row: CronTask): Promise<void> {
  await manageApi.cron.change({ id: row.id, status: !row.state })
  await load()
}

async function remove(row: CronTask): Promise<void> {
  if (!confirm(`确认删除任务「${row.name}」？`)) return
  await manageApi.cron.remove(row.id)
  await load()
}

onMounted(load)
</script>

<template>
  <ManageTable
    :columns="columns"
    :rows="rows"
    row-key="id"
    :loading="loading"
    empty="暂无定时任务"
  >
    <template #toolbar>
      <h2 class="text-lg font-[var(--gf-fw-semibold)]">定时任务</h2>
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
        {{ row.state ? '运行中' : '停止' }}
      </BaseTag>
      <code
        v-else-if="col.key === 'cron'"
        class="font-[var(--gf-font-mono)] text-[var(--gf-fs-xs)] text-[var(--gf-text-link)]"
      >
        {{ row.cron }}
      </code>
      <span v-else>{{ row[col.key] ?? '—' }}</span>
    </template>

    <template #actions="{ row }">
      <div class="flex gap-[var(--gf-space-1)] justify-end">
        <BaseButton variant="ghost" size="sm" @click="toggleState(row)">
          {{ row.state ? '停止' : '启动' }}
        </BaseButton>
        <BaseButton variant="ghost" size="sm" @click="openEdit(row)">编辑</BaseButton>
        <BaseButton variant="danger" size="sm" @click="remove(row)">删除</BaseButton>
      </div>
    </template>
  </ManageTable>

  <BaseDialog v-model:visible="dialogOpen" :title="editing ? '编辑任务' : '新增任务'">
    <div class="flex flex-col gap-[var(--gf-space-4)]">
      <ManageFormField label="任务名" required>
        <ManageInput v-model="form.name" placeholder="例如：每日采集" />
      </ManageFormField>
      <ManageFormField label="Cron 表达式" required hint="格式：秒 分 时 日 月 周（如 0 0 3 * * *）">
        <ManageInput v-model="form.cron" placeholder="0 0 3 * * *" />
      </ManageFormField>
      <ManageFormField label="任务类型">
        <select
          v-model="form.jobType"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option value="collect">采集</option>
          <option value="cleanup">清理</option>
          <option value="custom">自定义</option>
        </select>
      </ManageFormField>
      <ManageFormField label="备注">
        <ManageInput v-model="form.remark!" placeholder="可选" />
      </ManageFormField>
      <ManageFormField label="启用">
        <ManageSwitch v-model="form.state" />
      </ManageFormField>
    </div>
    <template #footer>
      <BaseButton variant="ghost" @click="dialogOpen = false">取消</BaseButton>
      <BaseButton variant="gradient" :loading="submitting" @click="submit">保存</BaseButton>
    </template>
  </BaseDialog>
</template>

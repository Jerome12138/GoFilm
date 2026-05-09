<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { manageApi } from '@/api'
import type { FilmClass } from '@/types/manage'
import ManageFormField from '@/components/manage/ManageFormField.vue'
import ManageInput from '@/components/manage/ManageInput.vue'
import ManageTextarea from '@/components/manage/ManageTextarea.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseImage from '@/components/base/BaseImage.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

const router = useRouter()
const tree = ref<FilmClass[]>([])
const submitting = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)

const form = reactive({
  name: '',
  enName: '',
  cid: 0,
  pid: 0,
  picture: '',
  year: '',
  area: '',
  director: '',
  actor: '',
  classTag: '',
  content: ''
})

onMounted(async () => {
  try {
    tree.value = await manageApi.film.classTree()
  } catch {
    /* 忽略 */
  }
})

async function handleUpload(e: Event): Promise<void> {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploading.value = true
  uploadProgress.value = 0
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await manageApi.file.upload(fd, (p) => {
      uploadProgress.value = p
    })
    form.picture = res.url
  } finally {
    uploading.value = false
  }
}

async function submit(): Promise<void> {
  submitting.value = true
  try {
    await manageApi.film.add({ ...form })
    router.push('/manage/film')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="bg-surface rounded-card shadow-card p-[var(--gf-space-6)] max-w-[960px]">
    <header class="mb-[var(--gf-space-5)]">
      <h2 class="text-lg font-[var(--gf-fw-semibold)]">新增影片</h2>
      <p class="text-sm text-muted">手动录入一部影片信息（采集源会自动同步，此页用于补录）</p>
    </header>

    <form
      class="grid grid-cols-1 md:grid-cols-2 gap-[var(--gf-space-5)]"
      @submit.prevent="submit"
    >
      <ManageFormField label="影片名" required>
        <ManageInput v-model="form.name" />
      </ManageFormField>
      <ManageFormField label="英文名">
        <ManageInput v-model="form.enName" />
      </ManageFormField>

      <ManageFormField label="顶级分类 (Pid)">
        <select
          v-model="form.pid"
          class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-md)] px-[var(--gf-space-3)] py-[var(--gf-space-3)]"
          data-focusable="true"
        >
          <option :value="0">请选择</option>
          <option v-for="t in tree" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
      </ManageFormField>
      <ManageFormField label="子分类 (Cid)">
        <ManageInput v-model="form.cid" type="number" />
      </ManageFormField>

      <ManageFormField label="海报">
        <div class="flex items-center gap-[var(--gf-space-3)]">
          <div class="w-[80px] h-[110px] rounded-[var(--gf-radius-md)] overflow-hidden bg-elevated">
            <BaseImage v-if="form.picture" :src="form.picture" alt="poster" ratio="3/4" />
            <div v-else class="w-full h-full flex-center text-muted text-xs">暂无</div>
          </div>
          <label class="flex flex-col gap-[var(--gf-space-2)]">
            <input type="file" accept="image/*" class="hidden" @change="handleUpload" />
            <BaseButton variant="ghost" size="sm" type="button" :loading="uploading">
              <BaseIcon name="upload" size="16px" />
              {{ uploading ? `上传中 ${Math.round(uploadProgress * 100)}%` : '选择图片' }}
            </BaseButton>
            <ManageInput v-model="form.picture" placeholder="或粘贴图片 URL" />
          </label>
        </div>
      </ManageFormField>

      <ManageFormField label="年份">
        <ManageInput v-model="form.year" placeholder="2024" />
      </ManageFormField>
      <ManageFormField label="地区">
        <ManageInput v-model="form.area" placeholder="中国大陆" />
      </ManageFormField>
      <ManageFormField label="导演">
        <ManageInput v-model="form.director" />
      </ManageFormField>
      <ManageFormField label="主演">
        <ManageInput v-model="form.actor" placeholder="逗号分隔" />
      </ManageFormField>
      <ManageFormField label="标签">
        <ManageInput v-model="form.classTag" placeholder="逗号分隔" />
      </ManageFormField>

      <ManageFormField label="剧情简介" class="md:col-span-2">
        <ManageTextarea v-model="form.content" :rows="6" />
      </ManageFormField>

      <div class="md:col-span-2 flex justify-end gap-[var(--gf-space-3)]">
        <BaseButton variant="ghost" type="button" @click="router.back()">取消</BaseButton>
        <BaseButton variant="gradient" type="submit" :loading="submitting">保存</BaseButton>
      </div>
    </form>
  </section>
</template>

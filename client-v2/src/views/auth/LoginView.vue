<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import ManageFormField from '@/components/manage/ManageFormField.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const form = reactive({ username: '', password: '' })
const showPwd = ref(false)
const loading = ref(false)
const errorMsg = ref('')

onMounted(() => {
  // 已登录访问 /login 直接跳走
  if (userStore.isLoggedIn) {
    const redirect = (route.query.redirect as string) || '/manage/index'
    void router.replace(redirect)
  }
})

async function handleLogin(): Promise<void> {
  errorMsg.value = ''
  if (!form.username.trim()) {
    errorMsg.value = '请输入用户名'
    return
  }
  if (!form.password) {
    errorMsg.value = '请输入密码'
    return
  }
  loading.value = true
  try {
    await userStore.login({ username: form.username.trim(), password: form.password })
    const redirect = (route.query.redirect as string) || '/manage/index'
    await router.replace(redirect)
  } catch (e: unknown) {
    errorMsg.value = e instanceof Error ? e.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex flex-col gap-[var(--gf-space-6)]">
    <div class="text-center">
      <h1
        class="text-[var(--gf-fs-2xl)] font-[var(--gf-fw-black)] tracking-tight text-brand-gradient"
      >
        GoFilm Manage
      </h1>
      <p class="mt-[var(--gf-space-2)] text-secondary text-sm">影视站点管理后台</p>
    </div>

    <form class="flex flex-col gap-[var(--gf-space-4)]" @submit.prevent="handleLogin">
      <ManageFormField label="用户名" required>
        <div class="relative">
          <BaseIcon
            name="user"
            class="absolute left-[var(--gf-space-3)] top-1/2 -translate-y-1/2 text-muted"
            size="18px"
          />
          <input
            v-model="form.username"
            type="text"
            class="w-full bg-elevated/80 text-primary border border-default rounded-[var(--gf-radius-full)] pl-[var(--gf-space-10)] pr-[var(--gf-space-4)] py-[var(--gf-space-3)] text-sm outline-none focus:border-strong focus:shadow-focus transition"
            placeholder="用户名 / 邮箱"
            autocomplete="username"
            data-focusable="true"
          />
        </div>
      </ManageFormField>

      <ManageFormField label="密码" required>
        <div class="relative">
          <BaseIcon
            name="lock"
            class="absolute left-[var(--gf-space-3)] top-1/2 -translate-y-1/2 text-muted"
            size="18px"
          />
          <input
            v-model="form.password"
            :type="showPwd ? 'text' : 'password'"
            class="w-full bg-elevated/80 text-primary border border-default rounded-[var(--gf-radius-full)] pl-[var(--gf-space-10)] pr-[var(--gf-space-10)] py-[var(--gf-space-3)] text-sm outline-none focus:border-strong focus:shadow-focus transition"
            placeholder="密码"
            autocomplete="current-password"
            data-focusable="true"
            @keydown.enter="handleLogin"
          />
          <button
            type="button"
            class="absolute right-[var(--gf-space-3)] top-1/2 -translate-y-1/2 text-muted hover:text-primary"
            data-focusable="true"
            @click="showPwd = !showPwd"
          >
            <BaseIcon :name="showPwd ? 'eye-off' : 'eye'" size="18px" />
          </button>
        </div>
      </ManageFormField>

      <p
        v-if="errorMsg"
        class="text-xs text-[var(--gf-danger)] text-center"
      >
        {{ errorMsg }}
      </p>

      <BaseButton variant="gradient" size="lg" :loading="loading" type="submit">
        登录
      </BaseButton>
      <BaseButton variant="ghost" size="lg" disabled>
        注册（暂未开放）
      </BaseButton>
    </form>

    <p
      v-if="route.query.redirect"
      class="text-muted text-xs text-center"
    >
      登录后将跳转至：{{ route.query.redirect }}
    </p>
  </div>
</template>

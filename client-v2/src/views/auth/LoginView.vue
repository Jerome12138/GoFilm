<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
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

const redirectTo = computed(() => {
  const r = route.query.redirect
  return typeof r === 'string' && r ? r : ''
})

/** 用户已登录 → 兜底跳转：优先 redirect，否则按 role 决定 */
function redirectAfterLogin(): string {
  if (redirectTo.value) return redirectTo.value
  return userStore.isAdmin ? '/manage/index' : '/index'
}

onMounted(async () => {
  if (!userStore.isLoggedIn) return
  // 旧 token 可能没拉过 info，先尝试取一次 role 再决定跳哪里
  if (!userStore.info) {
    try {
      await userStore.fetchInfo()
    } catch {
      // 拉不到（token 失效）：拦截器会清 token，留在登录页
      return
    }
  }
  await router.replace(redirectAfterLogin())
})

async function handleLogin(): Promise<void> {
  errorMsg.value = ''
  if (!form.username.trim()) {
    errorMsg.value = '请输入用户名 / 邮箱'
    return
  }
  if (!form.password) {
    errorMsg.value = '请输入密码'
    return
  }
  loading.value = true
  try {
    await userStore.login({
      username: form.username.trim(),
      password: form.password
    })
    // login 内部已 fetchInfo，role 就位
    await router.replace(redirectAfterLogin())
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
        登录 GoFilm
      </h1>
      <p class="mt-[var(--gf-space-2)] text-secondary text-sm">
        登录后可同步观看历史 / 收藏 / 进入后台管理
      </p>
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
            class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-full)] pl-[var(--gf-space-10)] pr-[var(--gf-space-4)] py-[var(--gf-space-3)] text-sm outline-none focus:border-strong focus:shadow-focus transition"
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
            class="w-full bg-elevated text-primary border border-default rounded-[var(--gf-radius-full)] pl-[var(--gf-space-10)] pr-[var(--gf-space-10)] py-[var(--gf-space-3)] text-sm outline-none focus:border-strong focus:shadow-focus transition"
            placeholder="密码"
            autocomplete="current-password"
            data-focusable="true"
            @keydown.enter="handleLogin"
          />
          <button
            type="button"
            class="absolute right-[var(--gf-space-3)] top-1/2 -translate-y-1/2 text-muted hover:text-primary"
            data-focusable="true"
            aria-label="切换密码可见性"
            @click="showPwd = !showPwd"
          >
            <BaseIcon :name="showPwd ? 'eye-off' : 'eye'" size="18px" />
          </button>
        </div>
      </ManageFormField>

      <p
        v-if="errorMsg"
        class="text-xs text-[var(--gf-danger)] text-center"
        role="alert"
      >
        {{ errorMsg }}
      </p>

      <BaseButton variant="gradient" size="lg" :loading="loading" type="submit">
        登录
      </BaseButton>
    </form>

    <!-- 注册指引：公共注册已下线，由管理员后台创建账号 -->
    <div
      class="rounded-[var(--gf-radius-md)] border border-subtle bg-elevated/60 p-[var(--gf-space-4)] text-center"
    >
      <p class="text-secondary text-sm leading-[var(--gf-lh-relaxed)]">
        注册功能已暂时下线。
      </p>
      <p class="text-muted text-xs mt-[var(--gf-space-2)]">
        如需账号请联系管理员开通；管理员可在后台「系统管理 → 用户管理」创建账号。
      </p>
    </div>

    <p
      v-if="redirectTo"
      class="text-muted text-xs text-center"
    >
      登录后将跳转至：{{ redirectTo }}
    </p>
  </div>
</template>

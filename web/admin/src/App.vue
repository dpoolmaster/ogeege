<script setup>
import { onMounted, ref } from 'vue'
import { api } from './api'
import HelpButton from './components/HelpButton.vue'

const user = ref(null)
const error = ref('')

onMounted(async () => {
  try {
    const data = await api('/me')
    user.value = data
  } catch (err) {
    error.value = err.message
  }
})

async function logout() {
  error.value = ''
  try {
    await api('/auth/logout', { method: 'POST' })
  } catch (err) {
    error.value = err.message
    return
  }
  window.location.href = '/login'
}
</script>

<template>
  <div v-if="error" class="admin-content">
    <p class="admin-error">{{ error }}</p>
    <a href="/login">Войти</a>
  </div>
  <div v-else-if="!user" class="admin-content">
    <p aria-busy="true">Загрузка...</p>
  </div>
  <div v-else class="admin-shell">
    <header class="admin-topbar">
      <strong class="admin-brand">Админка</strong>
      <div class="admin-user">
        <span>{{ user.email }} ({{ user.role }})</span>
        <button type="button" class="secondary admin-logout" @click="logout">Выйти</button>
      </div>
    </header>
    <div class="admin-layout">
      <nav class="admin-nav">
        <ul>
          <li><router-link to="/exams">Треки</router-link></li>
          <li><router-link to="/exam-years">Учебные годы</router-link></li>
          <li><router-link to="/kim-numbers">Номера КИМ</router-link></li>
          <li><router-link to="/topics">Темы</router-link></li>
          <li><router-link to="/theory">Теория</router-link></li>
          <li><router-link to="/tasks">Задания</router-link></li>
          <li><router-link to="/variants">Варианты</router-link></li>
          <li><router-link to="/rubrics">Рубрики</router-link></li>
          <li><router-link to="/score-scales">Шкалы</router-link></li>
          <li v-if="user.role === 'admin'"><router-link to="/users">Пользователи</router-link></li>
          <li v-if="user.role === 'admin'"><router-link to="/audit">Аудит</router-link></li>
        </ul>
      </nav>
      <main class="admin-content">
        <HelpButton />
        <router-view />
      </main>
    </div>
  </div>
</template>

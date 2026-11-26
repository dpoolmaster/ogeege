<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'

const route = useRoute()
const dialog = ref(null)
const title = ref('')
const html = ref('')
const error = ref('')
const loading = ref(false)

function sectionId() {
  const path = route.path.replace(/^\/+|\/+$/g, '')
  return path ? 'admin-' + path : ''
}

async function open() {
  title.value = ''
  html.value = ''
  error.value = ''
  loading.value = true
  dialog.value?.showModal()

  const id = sectionId()
  if (!id) {
    error.value = 'Справка для этого раздела недоступна'
    loading.value = false
    return
  }

  try {
    const data = await api('/help/' + id)
    title.value = data.title
    html.value = data.html
  } catch (err) {
    error.value = err.message || 'Не удалось загрузить справку'
  } finally {
    loading.value = false
  }
}

function close() {
  dialog.value?.close()
}
</script>

<template>
  <button
    type="button"
    class="admin-help-button"
    aria-label="Справка по разделу"
    title="Справка по разделу"
    @click="open"
  >
    ?
  </button>
  <dialog ref="dialog" class="admin-help-dialog">
    <article>
      <header>
        <strong>{{ title || 'Справка' }}</strong>
        <button type="button" class="admin-help-close" aria-label="Закрыть" @click="close">×</button>
      </header>
      <p v-if="loading" aria-busy="true">Загрузка...</p>
      <p v-else-if="error" class="admin-error">{{ error }}</p>
      <div v-else class="admin-help-body" v-html="html"></div>
    </article>
  </dialog>
</template>

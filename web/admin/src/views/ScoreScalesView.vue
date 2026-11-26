<script setup>
import { ref } from 'vue'
import { api } from '../api'

const examYearId = ref('')
const entries = ref('[]')
const error = ref('')

async function load() {
  error.value = ''
  try {
    const data = await api(`/admin/score-scales?exam_year_id=${examYearId.value}`)
    entries.value = JSON.stringify(data.items || [], null, 2)
  } catch (err) {
    error.value = err.message
  }
}

async function save() {
  error.value = ''
  try {
    await api(`/admin/score-scales/${examYearId.value}`, {
      method: 'PUT',
      body: JSON.stringify({ entries: JSON.parse(entries.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <h2>Шкалы перевода</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <input v-model="examYearId" placeholder="ID учебного года" required />
    <button type="submit">Загрузить</button>
  </form>
  <textarea v-model="entries" placeholder='[{"primary_score":0,"test_score":0,"grade":2,"min_threshold":null}]'></textarea>
  <button :disabled="!examYearId" @click="save">Сохранить</button>
</template>

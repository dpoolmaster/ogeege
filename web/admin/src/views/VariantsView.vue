<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const manualTasks = ref('[]')
const rules = ref('[]')
const form = reactive({ exam_year_id: '', title: 'Вариант', kind: 'mock', duration_minutes: 235 })

async function load() {
  try {
    const data = await api('/admin/variants?limit=100')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/variants', {
      method: 'POST',
      body: JSON.stringify({ ...form, tasks: JSON.parse(manualTasks.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function generate() {
  error.value = ''
  try {
    await api('/admin/variants/generate', {
      method: 'POST',
      body: JSON.stringify({ ...form, rules: JSON.parse(rules.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/variants/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Варианты</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <input v-model="form.exam_year_id" placeholder="ID учебного года" required />
    <input v-model="form.title" placeholder="Название" required />
    <select v-model="form.kind">
      <option value="diagnostic">Диагностика</option>
      <option value="practice">Практика</option>
      <option value="mock">Пробник</option>
    </select>
    <input v-model.number="form.duration_minutes" type="number" placeholder="Минуты" required />
    <textarea v-model="manualTasks" placeholder='[{"task_id":"...","points":1}]'></textarea>
    <button type="submit">Создать вручную</button>
  </form>
  <textarea v-model="rules" placeholder='[{"kim_number_id":"...","count":1,"difficulty":"easy","topic_id":null}]'></textarea>
  <button @click="generate">Сгенерировать</button>
  <table>
    <thead><tr><th>Название</th><th>Вид</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.title }}</td>
        <td>{{ item.kind }}</td>
        <td>{{ item.status }}</td>
        <td><button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button></td>
      </tr>
    </tbody>
  </table>
</template>

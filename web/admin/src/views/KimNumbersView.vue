<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const filter = reactive({ exam_year_id: '' })
const form = reactive({
  exam_year_id: '',
  number: 1,
  part: 1,
  answer_type: 'short',
  max_primary_score: 1,
  title: ''
})

async function load() {
  error.value = ''
  try {
    const query = filter.exam_year_id ? `?exam_year_id=${filter.exam_year_id}&limit=200` : '?limit=200'
    const data = await api('/admin/kim-numbers' + query)
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/kim-numbers', { method: 'POST', body: JSON.stringify(form) })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(item) {
  error.value = ''
  try {
    await api(`/admin/kim-numbers/${item.id}`, { method: 'DELETE' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Номера КИМ</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <input v-model="filter.exam_year_id" placeholder="ID учебного года" />
    <button type="submit">Фильтр</button>
  </form>
  <form class="admin-form" @submit.prevent="create">
    <input v-model="form.exam_year_id" placeholder="ID учебного года" required />
    <input v-model.number="form.number" type="number" placeholder="Номер" required />
    <input v-model.number="form.part" type="number" placeholder="Часть" required />
    <select v-model="form.answer_type">
      <option value="short">Краткий</option>
      <option value="extended">Развёрнутый</option>
    </select>
    <input v-model.number="form.max_primary_score" type="number" placeholder="Макс. балл" required />
    <input v-model="form.title" placeholder="Заголовок" />
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Год</th><th>Номер</th><th>Часть</th><th>Ответ</th><th>Балл</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.exam_year_id }}</td>
        <td>{{ item.number }}</td>
        <td>{{ item.part }}</td>
        <td>{{ item.answer_type }}</td>
        <td>{{ item.max_primary_score }}</td>
        <td><button class="secondary" @click="remove(item)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>

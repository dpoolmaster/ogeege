<script setup>
import { onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const error = ref('')
const editingId = ref('')
const criteria = ref('[]')
const form = reactive({ kim_number_id: '', exam_year_id: '' })

async function load() {
  try {
    const data = await api('/admin/rubrics?limit=200')
    items.value = data.items || []
  } catch (err) {
    error.value = err.message
  }
}

async function create() {
  error.value = ''
  try {
    await api('/admin/rubrics', {
      method: 'POST',
      body: JSON.stringify({ ...form, criteria: JSON.parse(criteria.value) })
    })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

function edit(item) {
  editingId.value = item.id
  criteria.value = JSON.stringify(item.criteria)
}

async function update() {
  error.value = ''
  try {
    await api(`/admin/rubrics/${editingId.value}`, {
      method: 'PUT',
      body: JSON.stringify({ criteria: JSON.parse(criteria.value) })
    })
    editingId.value = ''
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function publish(item) {
  error.value = ''
  try {
    await api(`/admin/rubrics/${item.id}/publish`, { method: 'POST' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

onMounted(load)
</script>

<template>
  <h2>Рубрики</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="create">
    <input v-model="form.kim_number_id" placeholder="ID номера КИМ" required />
    <input v-model="form.exam_year_id" placeholder="ID учебного года" required />
    <textarea v-model="criteria" placeholder='[{"code":"C1","description":"...","max_points":2}]'></textarea>
    <button type="submit">Создать</button>
  </form>
  <button v-if="editingId" @click="update">Сохранить новую версию</button>
  <table>
    <thead><tr><th>Версия</th><th>Номер КИМ</th><th>Статус</th><th></th></tr></thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ item.version }}</td>
        <td>{{ item.kim_number_id }}</td>
        <td>{{ item.status }}</td>
        <td class="admin-actions">
          <button @click="edit(item)">Изменить</button>
          <button v-if="item.status !== 'published'" @click="publish(item)">Опубликовать</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>

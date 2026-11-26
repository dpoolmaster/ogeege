<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { api } from '../api'

const items = ref([])
const examYears = ref([])
const exams = ref([])
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

const examTitles = computed(() => {
  const map = {}
  for (const exam of exams.value) map[exam.id] = exam.title
  return map
})

const examYearLabels = computed(() => {
  const map = {}
  for (const year of examYears.value) {
    const title = examTitles.value[year.exam_id]
    map[year.id] = title ? `${title} — ${year.year}` : String(year.year)
  }
  return map
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

async function loadRefs() {
  try {
    const [years, examList] = await Promise.all([
      api('/admin/exam-years?limit=200'),
      api('/admin/exams?limit=200')
    ])
    examYears.value = years.items || []
    exams.value = examList.items || []
    if (!form.exam_year_id && examYears.value.length > 0) form.exam_year_id = examYears.value[0].id
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

onMounted(() => {
  load()
  loadRefs()
})
</script>

<template>
  <h2>Номера КИМ</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <label>
      Учебный год
      <select v-model="filter.exam_year_id">
        <option value="">Все годы</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <button type="submit">Фильтр</button>
  </form>
  <form class="admin-form" @submit.prevent="create">
    <label>
      Учебный год
      <select v-model="form.exam_year_id" required>
        <option value="" disabled>Выберите год</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <label>
      Номер задания
      <input v-model.number="form.number" type="number" required />
    </label>
    <label>
      Часть
      <input v-model.number="form.part" type="number" required />
    </label>
    <label>
      Тип ответа
      <select v-model="form.answer_type">
        <option value="short">Краткий</option>
        <option value="extended">Развёрнутый</option>
      </select>
    </label>
    <label>
      Макс. первичный балл
      <input v-model.number="form.max_primary_score" type="number" required />
    </label>
    <label>
      Заголовок (необязательно)
      <input v-model="form.title" />
    </label>
    <button type="submit">Создать</button>
  </form>
  <table>
    <thead>
      <tr><th>Год</th><th>Номер</th><th>Часть</th><th>Ответ</th><th>Балл</th><th></th></tr>
    </thead>
    <tbody>
      <tr v-for="item in items" :key="item.id">
        <td>{{ examYearLabels[item.exam_year_id] || item.exam_year_id }}</td>
        <td>{{ item.number }}</td>
        <td>{{ item.part }}</td>
        <td>{{ item.answer_type }}</td>
        <td>{{ item.max_primary_score }}</td>
        <td><button class="secondary" @click="remove(item)">Удалить</button></td>
      </tr>
    </tbody>
  </table>
</template>

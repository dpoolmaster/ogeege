<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'

const examYears = ref([])
const exams = ref([])
const examYearId = ref('')
const entries = ref('[]')
const error = ref('')

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

async function loadRefs() {
  try {
    const [years, examList] = await Promise.all([
      api('/admin/exam-years?limit=200'),
      api('/admin/exams?limit=200')
    ])
    examYears.value = years.items || []
    exams.value = examList.items || []
    if (!examYearId.value && examYears.value.length > 0) examYearId.value = examYears.value[0].id
  } catch (err) {
    error.value = err.message
  }
}

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

onMounted(loadRefs)
</script>

<template>
  <h2>Шкалы перевода</h2>
  <p v-if="error" class="admin-error">{{ error }}</p>
  <form class="admin-form" @submit.prevent="load">
    <label>
      Учебный год
      <select v-model="examYearId" required>
        <option value="" disabled>Выберите год</option>
        <option v-for="year in examYears" :key="year.id" :value="year.id">{{ examYearLabels[year.id] }}</option>
      </select>
    </label>
    <button type="submit">Загрузить</button>
  </form>
  <div class="admin-field">
    <span>Строки шкалы (JSON)</span>
    <textarea v-model="entries" placeholder='[{"primary_score":0,"test_score":0,"grade":2,"min_threshold":null}]'></textarea>
  </div>
  <button :disabled="!examYearId" @click="save">Сохранить</button>
</template>

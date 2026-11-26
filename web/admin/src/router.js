import { createRouter, createWebHistory } from 'vue-router'
import ExamsView from './views/ExamsView.vue'
import ExamYearsView from './views/ExamYearsView.vue'
import KimNumbersView from './views/KimNumbersView.vue'
import TopicsView from './views/TopicsView.vue'
import TheoryView from './views/TheoryView.vue'
import TasksView from './views/TasksView.vue'
import VariantsView from './views/VariantsView.vue'
import RubricsView from './views/RubricsView.vue'
import ScoreScalesView from './views/ScoreScalesView.vue'
import UsersView from './views/UsersView.vue'
import AuditView from './views/AuditView.vue'

const routes = [
  { path: '/', redirect: '/exams' },
  { path: '/exams', component: ExamsView },
  { path: '/exam-years', component: ExamYearsView },
  { path: '/kim-numbers', component: KimNumbersView },
  { path: '/topics', component: TopicsView },
  { path: '/theory', component: TheoryView },
  { path: '/tasks', component: TasksView },
  { path: '/variants', component: VariantsView },
  { path: '/rubrics', component: RubricsView },
  { path: '/score-scales', component: ScoreScalesView },
  { path: '/users', component: UsersView },
  { path: '/audit', component: AuditView }
]

export default createRouter({
  history: createWebHistory('/admin'),
  routes
})

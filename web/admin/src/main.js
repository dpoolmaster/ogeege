import { createApp } from 'vue'
import '@picocss/pico/css/pico.min.css'
import 'katex/dist/katex.min.css'
import 'highlight.js/styles/github.css'
import './style.css'
import App from './App.vue'
import router from './router'

createApp(App).use(router).mount('#app')

<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { createLowlight, common } from 'lowlight'
import TurndownService from 'turndown'
import { gfm } from '@joplin/turndown-plugin-gfm'
import { marked } from 'marked'
import renderMathInElement from 'katex/contrib/auto-render'

const props = defineProps({
  modelValue: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue'])

const lowlight = createLowlight(common)
const turndown = new TurndownService({ headingStyle: 'atx', codeBlockStyle: 'fenced' })
turndown.use(gfm)
const preview = ref(null)

const editor = useEditor({
  content: marked.parse(props.modelValue || ''),
  extensions: [
    StarterKit.configure({ codeBlock: false }),
    CodeBlockLowlight.configure({ lowlight })
  ],
  onUpdate: ({ editor }) => {
    const markdown = turndown.turndown(editor.getHTML())
    emit('update:modelValue', markdown)
    renderPreview(markdown)
  }
})

watch(
  () => props.modelValue,
  (value) => {
    if (!editor.value) {
      return
    }
    const current = turndown.turndown(editor.value.getHTML())
    if (current !== value) {
      editor.value.commands.setContent(marked.parse(value || ''))
    }
    renderPreview(value)
  },
  { immediate: true }
)

function renderPreview(markdown) {
  nextTick(() => {
    if (!preview.value) {
      return
    }
    preview.value.innerHTML = marked.parse(markdown || '')
    renderMathInElement(preview.value, {
      delimiters: [
        { left: '$$', right: '$$', display: true },
        { left: '$', right: '$', display: false }
      ],
      throwOnError: false
    })
  })
}

function insertFormula() {
  const formula = window.prompt('Формула LaTeX', 'x^2')
  if (formula && editor.value) {
    editor.value.chain().focus().insertContent('$' + formula + '$').run()
  }
}

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<template>
  <div>
    <div v-if="editor" class="editor-toolbar">
      <button type="button" @click="editor.chain().focus().toggleBold().run()">Ж</button>
      <button type="button" @click="editor.chain().focus().toggleItalic().run()">К</button>
      <button type="button" @click="editor.chain().focus().toggleHeading({ level: 2 }).run()">H2</button>
      <button type="button" @click="editor.chain().focus().toggleBulletList().run()">Список</button>
      <button type="button" @click="editor.chain().focus().toggleCodeBlock().run()">Код</button>
      <button type="button" @click="insertFormula">Формула</button>
    </div>
    <div class="editor-grid">
      <EditorContent :editor="editor" class="editor-surface" />
      <div ref="preview" class="editor-preview"></div>
    </div>
  </div>
</template>

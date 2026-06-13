import React from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import ImageExt from '@tiptap/extension-image'
import { Bold, Italic, Heading, List, ListOrdered, Quote, Link as LinkIcon, Undo, Redo } from 'lucide-react'

interface RichEditorProps {
  value: string
  onChange: (html: string) => void
  placeholder?: string
  readOnly?: boolean
}

const MenuButton: React.FC<{ onClick: () => void; active?: boolean; title: string; children: React.ReactNode }> = ({ onClick, active, title, children }) => (
  <button type="button" onClick={onClick} title={title}
    className={`p-1.5 rounded hover:bg-slate-100 ${active ? 'bg-slate-200 text-blue-600' : 'text-slate-600'}`}>
    {children}
  </button>
)

export const RichEditor: React.FC<RichEditorProps> = ({ value, onChange, placeholder, readOnly }) => {
  const editor = useEditor({
    extensions: [StarterKit, Link, ImageExt],
    content: value,
    onUpdate: ({ editor }) => onChange(editor.getHTML()),
    editorProps: { attributes: { class: 'prose prose-sm max-w-none focus:outline-none min-h-[150px] px-3 py-2' } },
  })!

  if (!editor) return null

  return (
    <div className="border border-gray-300 rounded-lg overflow-hidden">
      {!readOnly && (
        <div className="flex flex-wrap gap-0.5 p-1.5 border-b bg-gray-50">
          <MenuButton onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive('bold')} title="加粗">
            <Bold className="w-4 h-4" />
          </MenuButton>
          <MenuButton onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive('italic')} title="斜体">
            <Italic className="w-4 h-4" />
          </MenuButton>
          <span className="w-px h-5 bg-gray-300 mx-1 self-center" />
          <MenuButton onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive('heading', { level: 2 })} title="标题">
            <Heading className="w-4 h-4" />
          </MenuButton>
          <span className="w-px h-5 bg-gray-300 mx-1 self-center" />
          <MenuButton onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive('bulletList')} title="无序列表">
            <List className="w-4 h-4" />
          </MenuButton>
          <MenuButton onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive('orderedList')} title="有序列表">
            <ListOrdered className="w-4 h-4" />
          </MenuButton>
          <MenuButton onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} title="引用">
            <Quote className="w-4 h-4" />
          </MenuButton>
          <span className="w-px h-5 bg-gray-300 mx-1 self-center" />
          <MenuButton onClick={() => { const url = prompt('输入链接'); if (url) editor.chain().focus().setLink({ href: url }).run() }} active={editor.isActive('link')} title="链接">
            <LinkIcon className="w-4 h-4" />
          </MenuButton>
          <span className="w-px h-5 bg-gray-300 mx-1 self-center" />
          <MenuButton onClick={() => editor.chain().focus().undo().run()} title="撤销">
            <Undo className="w-4 h-4" />
          </MenuButton>
          <MenuButton onClick={() => editor.chain().focus().redo().run()} title="重做">
            <Redo className="w-4 h-4" />
          </MenuButton>
        </div>
      )}
      <EditorContent editor={editor} />
    </div>
  )
}

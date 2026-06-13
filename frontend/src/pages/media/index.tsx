import React, { useEffect, useState } from 'react'
import { Image, File, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { Media } from '../../types'

export const MediaPage: React.FC = () => {
  const [items, setItems] = useState<Media[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(false)
  const pageSize = 20

  const fetchList = async () => {
    setLoading(true)
    try {
      const params = `page=${page}&page_size=${pageSize}${type ? '&type=' + type : ''}`
      const res = await api.get<{ data: Media[]; total: number }>('/media?' + params)
      setItems(res.data)
      setTotal(res.total)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchList() }, [page, type])

  const handleDelete = async (id: number) => {
    if (!confirm('确认删除该文件？')) return
    await api.del('/media/' + id)
    fetchList()
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const form = new FormData()
    form.append('file', file)
    const token = localStorage.getItem('auth_token')
    await fetch('/api/v1/media/upload', {
      method: 'POST',
      headers: token ? { Authorization: 'Bearer ' + token } : {},
      body: form,
    })
    e.target.value = ''
    fetchList()
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">媒体管理</h2>
        <label className="px-4 py-2 bg-blue-600 text-white rounded-lg cursor-pointer hover:bg-blue-700 text-sm">
          上传文件
          <input type="file" className="hidden" onChange={handleUpload} />
        </label>
      </div>

      <div className="flex gap-2">
        {['', 'image', 'file'].map(t => (
          <button key={t} onClick={() => { setType(t); setPage(1) }}
            className={`px-3 py-1.5 rounded-lg text-sm ${type === t ? 'bg-blue-600 text-white' : 'bg-white border text-slate-600 hover:bg-slate-50'}`}>
            {t || '全部'}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="text-center text-slate-400 py-12">加载中...</div>
      ) : items.length === 0 ? (
        <div className="text-center text-slate-400 py-12">暂无文件</div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
          {items.map(item => (
            <div key={item.id} className="bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden group">
              <div className="h-32 bg-slate-100 flex items-center justify-center">
                {item.type === 'image' ? (
                  <img src={item.url} alt={item.original_name} className="w-full h-full object-cover" />
                ) : (
                  <File className="w-10 h-10 text-slate-400" />
                )}
              </div>
              <div className="p-2">
                <p className="text-xs text-slate-600 truncate">{item.original_name}</p>
                <p className="text-xs text-slate-400 mt-1">{(item.size / 1024).toFixed(1)} KB</p>
                <button onClick={() => handleDelete(item.id)}
                  className="mt-1 text-xs text-red-500 hover:text-red-700 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Trash2 className="w-3.5 h-3.5 inline" /> 删除
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-4">
          {Array.from({ length: totalPages }, (_, i) => i + 1).map(p => (
            <button key={p} onClick={() => setPage(p)}
              className={`px-3 py-1 rounded text-sm ${p === page ? 'bg-blue-600 text-white' : 'bg-white border text-slate-600'}`}>
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

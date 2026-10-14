'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api, type ReraDefinition, type ReraForm } from '@/lib/api'
import ReraFormView from '@/components/rera/ReraFormView'

export default function ReraFormPrintPage() {
  const { id } = useParams<{ id: string }>()
  const [form, setForm] = useState<ReraForm | null>(null)
  const [def, setDef] = useState<ReraDefinition | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api.rera.get(id).then(r => { setForm(r.form); setDef(r.definition) })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load'))
  }, [id])

  if (error) return <p role="alert" className="p-8 text-sm text-red-600">{error}</p>
  if (!form || !def) return <p className="p-8 text-sm text-gray-400">Loading… جارٍ التحميل</p>

  return (
    <div className="flex-1 overflow-y-auto bg-gray-100 print:bg-white print:overflow-visible">
      <div className="max-w-[210mm] mx-auto my-6 print:my-0 bg-white shadow print:shadow-none p-10 print:p-0">
        <div className="flex justify-end mb-4 print:hidden">
          <button onClick={() => window.print()} className="px-4 py-2 text-sm bg-gray-900 text-white rounded-lg">Print / Save as PDF · طباعة</button>
        </div>
        <ReraFormView definition={def} data={form.data} />
        <div className="grid grid-cols-2 gap-10 mt-10">
          {def.signers.map(s => (
            <div key={s} className="border-t border-gray-400 pt-1 text-xs text-gray-600">{s.replace('_', ' ')} — signature · التوقيع</div>
          ))}
        </div>
      </div>
    </div>
  )
}

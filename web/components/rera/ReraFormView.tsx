import type { ReraDefinition } from '@/lib/api'

/** Renders a RERA form's fields bilingually (English | Arabic), for screen, print and signing. */
export default function ReraFormView({ definition, data }: { definition: ReraDefinition; data: Record<string, string> }) {
  return (
    <div className="text-sm text-gray-900">
      <h2 className="text-lg font-bold flex flex-wrap justify-between gap-2 mb-4">
        <span>{definition.title_en}</span>
        <span dir="rtl" lang="ar">{definition.title_ar}</span>
      </h2>
      {definition.sections.map(s => (
        <section key={s.en} className="mb-4 break-inside-avoid">
          <h3 className="flex justify-between text-xs font-semibold uppercase tracking-wide text-gray-500 border-b pb-1 mb-2">
            <span>{s.en}</span><span dir="rtl" lang="ar">{s.ar}</span>
          </h3>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1.5">
            {s.fields.map(f => (
              <div key={f.key} className={f.long ? 'sm:col-span-2' : ''}>
                <dt className="text-xs text-gray-500 flex justify-between gap-2">
                  <span>{f.en}</span><span dir="rtl" lang="ar">{f.ar}</span>
                </dt>
                <dd className={'min-h-[1.5rem] border-b border-dotted border-gray-300 ' + (f.long ? 'whitespace-pre-line' : '')}>
                  {data[f.key] || ''}
                </dd>
              </div>
            ))}
          </dl>
        </section>
      ))}
      <p className="text-[11px] text-gray-500 mt-4">
        Pre-filled by Masaar CRM from the brokerage&apos;s records for review and signature. It does not replace the official
        DLD/RERA forms or registration. · نموذج معبأ مسبقاً من سجلات الوسيط للمراجعة والتوقيع، ولا يغني عن النماذج والتسجيل الرسمي لدى دائرة الأراضي والأملاك.
      </p>
    </div>
  )
}

'use client'
import { useLang } from '@/context/LangContext'
import clsx from 'clsx'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  const { t } = useLang()

  if (totalPages <= 1) return null

  return (
    <div className="flex justify-center items-center gap-2 mt-8">
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page === 1}
        className={clsx(
          "px-3.5 py-2 text-xs font-medium rounded-xl border bg-white transition-colors",
          page === 1
            ? "border-surface-100 text-surface-300 cursor-not-allowed"
            : "border-surface-200 text-surface-700 hover:bg-surface-50"
        )}
      >
        {t('السابق', 'Previous')}
      </button>

      <span className="px-3 py-1.5 text-xs text-surface-500 font-medium tabular-nums">
        {page} / {totalPages}
      </span>

      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page === totalPages}
        className={clsx(
          "px-3.5 py-2 text-xs font-medium rounded-xl border bg-white transition-colors",
          page === totalPages
            ? "border-surface-100 text-surface-300 cursor-not-allowed"
            : "border-surface-200 text-surface-700 hover:bg-surface-50"
        )}
      >
        {t('التالي', 'Next')}
      </button>
    </div>
  )
}

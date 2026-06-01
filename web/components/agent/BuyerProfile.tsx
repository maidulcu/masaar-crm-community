'use client'
import { useState } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'

interface BuyerProfileData {
  areas?: string[]
  property_type?: string
  bedrooms?: string
  budget_min_aed?: number | null
  budget_max_aed?: number | null
  timeline?: string
  buyer_type?: string
  transaction_type?: string
  notes?: string
  confidence?: string
}

interface Props {
  threadId: string
}

const CONFIDENCE_COLORS: Record<string, string> = {
  high: 'bg-green-100 text-green-700',
  medium: 'bg-amber-100 text-amber-700',
  low: 'bg-gray-100 text-gray-600',
}

const BUYER_TYPE_LABELS: Record<string, { ar: string; en: string }> = {
  investor: { ar: 'مستثمر', en: 'Investor' },
  end_user: { ar: 'مستخدم نهائي', en: 'End-User' },
  unknown: { ar: 'غير محدد', en: 'Unknown' },
}

const TIMELINE_LABELS: Record<string, { ar: string; en: string }> = {
  immediate: { ar: 'فوري', en: 'Immediate' },
  '1-3 months': { ar: '1-3 أشهر', en: '1–3 months' },
  '3-6 months': { ar: '3-6 أشهر', en: '3–6 months' },
  flexible: { ar: 'مرن', en: 'Flexible' },
  unknown: { ar: 'غير محدد', en: 'Unknown' },
}

export function BuyerProfile({ threadId }: Props) {
  const { lang, t } = useLang()
  const [profile, setProfile] = useState<BuyerProfileData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [extracted, setExtracted] = useState(false)

  const handleExtract = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await api.ai.extractBuyerProfile(threadId) as BuyerProfileData
      setProfile(res)
      setExtracted(true)
    } catch {
      setError(t('فشل استخراج الملف الشخصي', 'Failed to extract buyer profile'))
    } finally {
      setLoading(false)
    }
  }

  const formatBudget = (min?: number | null, max?: number | null) => {
    if (!min && !max) return t('غير محدد', 'Not specified')
    const fmt = (n: number) =>
      new Intl.NumberFormat('en-AE', { style: 'currency', currency: 'AED', maximumFractionDigits: 0 }).format(n)
    if (min && max) return `${fmt(min)} – ${fmt(max)}`
    if (min) return `${t('من', 'From')} ${fmt(min)}`
    return `${t('حتى', 'Up to')} ${fmt(max!)}`
  }

  if (!extracted) {
    return (
      <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 text-center">
        <p className="text-xs text-blue-600 mb-3">
          {t('استخرج متطلبات المشتري من المحادثة تلقائياً', 'Auto-extract buyer requirements from the conversation')}
        </p>
        <button
          onClick={handleExtract}
          disabled={loading}
          className="px-4 py-2 bg-blue-600 text-white text-xs font-semibold rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center gap-2 mx-auto"
        >
          {loading && (
            <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
          )}
          {loading
            ? t('جاري التحليل...', 'Analyzing...')
            : t('استخراج ملف المشتري', 'Extract Buyer Profile')}
        </button>
        {error && <p className="text-xs text-red-500 mt-2">{error}</p>}
      </div>
    )
  }

  if (!profile) return null

  const confidence = profile.confidence || 'low'
  const buyerType = profile.buyer_type || 'unknown'
  const timeline = profile.timeline || 'unknown'

  return (
    <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-indigo-600 px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-white text-sm">🎯</span>
          <p className="text-white font-semibold text-sm">{t('ملف المشتري', 'Buyer Profile')}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full ${CONFIDENCE_COLORS[confidence]}`}>
            {t(
              confidence === 'high' ? 'ثقة عالية' : confidence === 'medium' ? 'ثقة متوسطة' : 'ثقة منخفضة',
              confidence === 'high' ? 'High confidence' : confidence === 'medium' ? 'Medium confidence' : 'Low confidence'
            )}
          </span>
          <button
            onClick={handleExtract}
            disabled={loading}
            className="text-[10px] text-blue-200 hover:text-white transition-colors disabled:opacity-50"
          >
            {loading ? '⏳' : t('تحديث', 'Refresh')}
          </button>
        </div>
      </div>

      <div className="p-4 space-y-3">
        {/* Areas */}
        {profile.areas && profile.areas.length > 0 && (
          <div>
            <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-wide mb-1">
              {t('المناطق المفضلة', 'Preferred Areas')}
            </p>
            <div className="flex flex-wrap gap-1">
              {profile.areas.map((area) => (
                <span key={area} className="px-2 py-0.5 bg-blue-50 text-blue-700 text-xs rounded-full font-medium">
                  {area}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Property details grid */}
        <div className="grid grid-cols-2 gap-2">
          {profile.property_type && profile.property_type !== 'any' && (
            <div className="bg-gray-50 rounded-lg p-2">
              <p className="text-[10px] text-gray-400 mb-0.5">{t('نوع العقار', 'Type')}</p>
              <p className="text-xs font-semibold text-gray-800 capitalize">{profile.property_type}</p>
            </div>
          )}
          {profile.bedrooms && profile.bedrooms !== 'any' && (
            <div className="bg-gray-50 rounded-lg p-2">
              <p className="text-[10px] text-gray-400 mb-0.5">{t('الغرف', 'Bedrooms')}</p>
              <p className="text-xs font-semibold text-gray-800">{profile.bedrooms}</p>
            </div>
          )}
          <div className="bg-gray-50 rounded-lg p-2 col-span-2">
            <p className="text-[10px] text-gray-400 mb-0.5">{t('الميزانية', 'Budget')}</p>
            <p className="text-xs font-semibold text-gray-800">
              {formatBudget(profile.budget_min_aed, profile.budget_max_aed)}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-2">
            <p className="text-[10px] text-gray-400 mb-0.5">{t('الجدول الزمني', 'Timeline')}</p>
            <p className="text-xs font-semibold text-gray-800">
              {lang === 'ar'
                ? TIMELINE_LABELS[timeline]?.ar ?? timeline
                : TIMELINE_LABELS[timeline]?.en ?? timeline}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-2">
            <p className="text-[10px] text-gray-400 mb-0.5">{t('نوع المشتري', 'Buyer Type')}</p>
            <p className="text-xs font-semibold text-gray-800">
              {lang === 'ar'
                ? BUYER_TYPE_LABELS[buyerType]?.ar ?? buyerType
                : BUYER_TYPE_LABELS[buyerType]?.en ?? buyerType}
            </p>
          </div>
        </div>

        {/* Notes */}
        {profile.notes && (
          <div className="bg-amber-50 border border-amber-100 rounded-lg p-2.5">
            <p className="text-[10px] font-semibold text-amber-600 mb-0.5">{t('ملاحظات', 'Notes')}</p>
            <p className="text-xs text-gray-700">{profile.notes}</p>
          </div>
        )}
      </div>
    </div>
  )
}

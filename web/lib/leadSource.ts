// Lead source labels, shared by the pipeline, the lead-source report and settings.

type Translate = (ar: string, en: string) => string

const LABELS: Record<string, [string, string]> = {
  whatsapp: ['واتساب', 'WhatsApp'],
  web: ['الموقع', 'Website'],
  referral: ['إحالة', 'Referral'],
  event: ['فعالية', 'Event'],
  bos24: ['BOS24', 'BOS24'],
  property_finder: ['بروبرتي فايندر', 'Property Finder'],
  bayut: ['بيوت', 'Bayut'],
  dubizzle: ['دوبيزل', 'Dubizzle'],
  portal: ['بوابة عقارية', 'Other portal'],
}

export function leadSourceLabel(source: string, t: Translate): string {
  const l = LABELS[source]
  return l ? t(l[0], l[1]) : source
}

// Sources a user can pick when creating a lead by hand.
export const MANUAL_SOURCES = ['whatsapp', 'web', 'referral', 'event', 'property_finder', 'bayut', 'dubizzle', 'portal']

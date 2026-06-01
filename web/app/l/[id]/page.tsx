import type { Metadata } from 'next'

const BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface PublicListingData {
  listing: {
    id: string; title: string; description: string
    property_type: string; listing_type: string
    price: number; currency: string; rent_period?: string
    bedrooms: number; bathrooms: number; total_sqft: number; parking_spaces: number
    furnishing?: string; year_built?: number; amenities?: string[]
    area: string; community: string; city: string; emirate: string
    cover_image_url: string; image_urls?: string[]
    status: string; reference_number: string; available_from?: string
  }
  company: {
    name: string; phone: string; email: string
    logo_url: string; primary_color: string
  }
}

async function getData(id: string): Promise<PublicListingData | null> {
  try {
    const res = await fetch(`${BASE}/api/public/listings/${id}`, {
      next: { revalidate: 300 }, // cache 5 min
    })
    if (!res.ok) return null
    return res.json()
  } catch {
    return null
  }
}

// ── Open Graph / SEO metadata ─────────────────────────────────────────────────
export async function generateMetadata({ params }: { params: { id: string } }): Promise<Metadata> {
  const data = await getData(params.id)
  if (!data) return { title: 'Listing Not Found' }

  const { listing, company } = data
  const price = `${listing.currency} ${listing.price.toLocaleString()}`
  const loc = [listing.area, listing.city].filter(Boolean).join(', ')
  const desc = `${listing.bedrooms}BR ${listing.property_type} in ${loc} — ${price}${listing.rent_period ? '/' + listing.rent_period : ''}. ${listing.description?.slice(0, 120) ?? ''}`

  return {
    title: `${listing.title} | ${company.name}`,
    description: desc,
    openGraph: {
      title: listing.title,
      description: desc,
      images: listing.cover_image_url ? [{ url: listing.cover_image_url, width: 1200, height: 630 }] : [],
      type: 'article',
    },
    twitter: {
      card: 'summary_large_image',
      title: listing.title,
      description: desc,
      images: listing.cover_image_url ? [listing.cover_image_url] : [],
    },
  }
}

// ── Page component ────────────────────────────────────────────────────────────
export default async function PublicListingPage({ params }: { params: { id: string } }) {
  const data = await getData(params.id)

  if (!data) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <p className="text-2xl font-bold text-gray-400 mb-2">404</p>
          <p className="text-gray-500">This listing is no longer available.</p>
        </div>
      </div>
    )
  }

  const { listing, company } = data
  const primaryColor = company.primary_color || '#1a3a5c'
  const priceStr = `${listing.currency} ${listing.price.toLocaleString()}${listing.rent_period ? ' / ' + listing.rent_period : ''}`
  const location = [listing.area, listing.community, listing.city, listing.emirate].filter(Boolean).join(', ')

  const specs = [
    { label: 'Bedrooms', value: listing.bedrooms },
    { label: 'Bathrooms', value: listing.bathrooms },
    { label: 'Size', value: `${listing.total_sqft?.toLocaleString()} sqft` },
    { label: 'Parking', value: listing.parking_spaces },
    ...(listing.furnishing ? [{ label: 'Furnishing', value: listing.furnishing }] : []),
    ...(listing.year_built ? [{ label: 'Year Built', value: listing.year_built }] : []),
  ]

  return (
    <div className="min-h-screen bg-gray-50 font-sans">
      {/* Nav */}
      <header style={{ backgroundColor: primaryColor }} className="text-white px-6 py-4 flex items-center justify-between">
        {company.logo_url
          ? <img src={company.logo_url} alt={company.name} className="h-8 object-contain" />
          : <span className="text-lg font-bold">{company.name}</span>
        }
        <a href={`tel:${company.phone}`} className="text-sm opacity-90 hover:opacity-100">{company.phone}</a>
      </header>

      {/* Cover image */}
      {listing.cover_image_url && (
        <div className="w-full h-72 md:h-96 overflow-hidden">
          <img src={listing.cover_image_url} alt={listing.title} className="w-full h-full object-cover" />
        </div>
      )}

      <main className="max-w-4xl mx-auto px-4 py-8 space-y-8">

        {/* Title + price */}
        <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span style={{ backgroundColor: primaryColor }} className="text-white text-xs font-bold px-3 py-1 rounded-full uppercase">
                {listing.listing_type === 'rent' ? 'For Rent' : 'For Sale'}
              </span>
              <span className="text-xs text-gray-500 capitalize">{listing.property_type}</span>
            </div>
            <h1 className="text-2xl md:text-3xl font-bold text-gray-900">{listing.title}</h1>
            {location && <p className="text-gray-500 mt-1 flex items-center gap-1">📍 {location}</p>}
          </div>
          <div className="text-right shrink-0">
            <p className="text-3xl font-bold" style={{ color: primaryColor }}>{priceStr}</p>
            {listing.reference_number && (
              <p className="text-xs text-gray-400 mt-1">Ref: {listing.reference_number}</p>
            )}
          </div>
        </div>

        {/* Specs grid */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
          {specs.map(s => (
            <div key={s.label} className="bg-white rounded-xl border border-gray-200 p-3 text-center">
              <p className="text-[10px] text-gray-400 uppercase tracking-wide">{s.label}</p>
              <p className="text-lg font-bold text-gray-800 mt-0.5">{s.value ?? '—'}</p>
            </div>
          ))}
        </div>

        {/* Description */}
        {listing.description && (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-800 mb-3">Description</h2>
            <p className="text-gray-600 leading-relaxed whitespace-pre-line">{listing.description}</p>
          </div>
        )}

        {/* Amenities */}
        {listing.amenities && listing.amenities.length > 0 && (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-800 mb-3">Amenities</h2>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
              {listing.amenities.map(a => (
                <div key={a} className="flex items-center gap-2 text-sm text-gray-600">
                  <span style={{ color: primaryColor }}>✓</span> {a}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Image gallery */}
        {listing.image_urls && listing.image_urls.length > 1 && (
          <div>
            <h2 className="text-lg font-semibold text-gray-800 mb-3">Gallery</h2>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {listing.image_urls.slice(0, 6).map((url, i) => (
                <img key={i} src={url} alt={`Photo ${i+1}`}
                  className="w-full h-40 object-cover rounded-xl border border-gray-200" />
              ))}
            </div>
          </div>
        )}

        {/* Contact CTA */}
        <div style={{ backgroundColor: primaryColor }} className="rounded-2xl p-6 text-white text-center">
          <h2 className="text-xl font-bold mb-1">Interested in this property?</h2>
          <p className="opacity-80 mb-4">Contact {company.name} for a viewing or more information.</p>
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <a href={`tel:${company.phone}`}
              className="bg-white font-semibold py-3 px-6 rounded-xl hover:bg-gray-100 transition-colors"
              style={{ color: primaryColor }}>
              📞 Call Now
            </a>
            <a href={`mailto:${company.email}?subject=Enquiry: ${listing.title}&body=I am interested in listing ${listing.reference_number}`}
              className="border border-white/50 text-white font-semibold py-3 px-6 rounded-xl hover:bg-white/10 transition-colors">
              ✉️ Send Email
            </a>
          </div>
        </div>
      </main>

      <footer className="text-center text-xs text-gray-400 py-6 border-t border-gray-200">
        {company.name} · {company.email}
      </footer>
    </div>
  )
}

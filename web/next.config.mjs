/** @type {import('next').NextConfig} */
// Security headers for every response.
//
// script-src keeps 'unsafe-inline' because Next.js injects inline bootstrap/hydration scripts and a
// nonce-based policy would force every page to render dynamically. The CSP therefore does not stop
// injected inline script by itself; its job here is to confine where scripts can load from, where
// data can be sent (connect-src: this site, the API and the WebSocket only) and to block
// clickjacking, <base>/<form> hijacking and plugin content. XSS is mitigated at the source
// (safeUrl, no dangerouslySetInnerHTML) and by keeping credentials out of reach: the access token is
// memory-only and the refresh token is an HttpOnly cookie.
const origin = (u) => {
  try { return new URL(u).origin } catch { return '' }
}
const apiOrigin = origin(process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080')
const wsOrigin = origin((process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080').replace(/^ws/, 'http'))
  .replace(/^http/, 'ws')
const isDev = process.env.NODE_ENV !== 'production'

const csp = [
  "default-src 'self'",
  // 'unsafe-eval' is needed by React's dev tooling only.
  `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ''} https://challenges.cloudflare.com`,
  "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
  "font-src 'self' data: https://fonts.gstatic.com",
  // Listing photos come from arbitrary https hosts; map tiles/markers from OSM and cdnjs.
  "img-src 'self' data: blob: https:",
  `connect-src 'self' ${[apiOrigin, wsOrigin].filter(Boolean).join(' ')}${isDev ? ' ws://localhost:3000' : ''}`,
  'frame-src https://challenges.cloudflare.com',
  "frame-ancestors 'none'",
  "base-uri 'self'",
  "object-src 'none'",
  "form-action 'self'",
].join('; ')

const securityHeaders = [
  { key: 'Content-Security-Policy', value: csp },
  { key: 'X-Frame-Options', value: 'DENY' },
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  { key: 'Permissions-Policy', value: 'camera=(), microphone=(), payment=(), usb=()' },
  { key: 'Cross-Origin-Opener-Policy', value: 'same-origin' },
]

import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const nextConfig = {
  output: 'standalone',
  poweredByHeader: false,
  webpack: (config) => {
    config.resolve.alias['@'] = __dirname
    return config
  },
  async headers() {
    return [{ source: '/:path*', headers: securityHeaders }]
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/:path*`,
      },
      {
        source: '/webhooks/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/webhooks/:path*`,
      },
    ]
  },
}

export default nextConfig

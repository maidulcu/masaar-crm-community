import type { Metadata } from 'next'
import './globals.css'
import { LangProvider } from '@/context/LangContext'
import { ToastProvider } from '@/components/ui/Toast'
import { ErrorBoundary } from '@/components/ui/ErrorBoundary'
import PWARegister from '@/components/PWARegister'

export const metadata: Metadata = {
  title: 'Masaar CRM',
  description: 'The open-source CRM built for the UAE market',
  manifest: '/manifest.json',
  themeColor: '#2563eb',
  appleWebApp: {
    capable: true,
    statusBarStyle: 'default',
    title: 'Masaar CRM',
  },
  viewport: 'width=device-width, initial-scale=1, viewport-fit=cover',
  icons: {
    icon: '/icons/icon-192.svg',
    apple: '/icons/icon-512.svg',
  },
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // suppressHydrationWarning because LangContext updates dir/lang client-side
    // based on localStorage; the default matches the UAE-first Arabic default.
    <html lang="ar" dir="rtl" suppressHydrationWarning>
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Cairo:wght@400;500;600;700&family=Inter:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
        <link rel="apple-touch-icon" href="/icons/icon-512.svg" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content="Masaar CRM" />
        <meta name="mobile-web-app-capable" content="yes" />
      </head>
      <body className="bg-surface-50 text-surface-900 antialiased">
        <ErrorBoundary>
          <LangProvider>
            <ToastProvider>
              {children}
            </ToastProvider>
          </LangProvider>
        </ErrorBoundary>
        <PWARegister />
      </body>
    </html>
  )
}

import Link from 'next/link'
import { auth0 } from '@/lib/auth0';
import { redirect } from 'next/navigation';
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/app-sidebar'
import DashboardContent from '@/components/dashboard-content';

export default async function HomePage() {
  const session = await auth0.getSession();
  const user = session?.user;

  // If user is authenticated, show dashboard
  if (user) {
    return (
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="flex flex-col h-screen overflow-hidden">
          <div className="flex flex-1 flex-col p-4 overflow-auto">
            <DashboardContent />
          </div>
        </SidebarInset>
      </SidebarProvider>
    );
  }

  // Landing page for unauthenticated users
  return (
    <div className="min-h-screen bg-gradient-to-b from-blue-50 to-white dark:from-gray-900 dark:to-gray-800">
      {/* Navigation */}
      <nav className="border-b bg-white/80 backdrop-blur-sm dark:bg-gray-900/80">
        <div className="container mx-auto flex h-16 items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <svg className="h-8 w-8 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
            </svg>
            <span className="text-xl font-bold">URL Shortener</span>
          </div>
          <div className="flex items-center gap-4">
            <a href="/auth/login?returnTo=/" className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 transition-colors">
              Sign In
            </a>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <main className="container mx-auto px-4 py-20">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-5xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-6xl">
            Shorten Links,{' '}
            <span className="text-blue-600">Track Everything</span>
          </h1>
          <p className="mt-6 text-lg text-gray-600 dark:text-gray-300">
            Create short, memorable links and get detailed analytics on every click.
            See who's clicking, where they're from, and what devices they use.
          </p>
          <div className="mt-10 flex items-center justify-center gap-4">
            <a 
              href="/auth/login?returnTo=/"
              className="rounded-lg bg-blue-600 px-6 py-3 text-lg font-semibold text-white shadow-lg hover:bg-blue-700 transition-all hover:shadow-xl"
            >
              Get Started Free
            </a>
            <Link 
              href="#features"
              className="rounded-lg border border-gray-300 px-6 py-3 text-lg font-semibold text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800 transition-colors"
            >
              Learn More
            </Link>
          </div>
        </div>

        {/* Features Section */}
        <div id="features" className="mt-32 grid gap-8 md:grid-cols-3">
          <div className="rounded-xl border bg-white p-6 shadow-sm dark:bg-gray-800 dark:border-gray-700">
            <div className="mb-4 inline-flex rounded-lg bg-blue-100 p-3 dark:bg-blue-900">
              <svg className="h-6 w-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold">Lightning Fast</h3>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Sub-50ms redirects with 95%+ cache hit rate. Your links are always fast.
            </p>
          </div>
          <div className="rounded-xl border bg-white p-6 shadow-sm dark:bg-gray-800 dark:border-gray-700">
            <div className="mb-4 inline-flex rounded-lg bg-green-100 p-3 dark:bg-green-900">
              <svg className="h-6 w-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold">Detailed Analytics</h3>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Track clicks, devices, browsers, geographic locations, and referrers.
            </p>
          </div>
          <div className="rounded-xl border bg-white p-6 shadow-sm dark:bg-gray-800 dark:border-gray-700">
            <div className="mb-4 inline-flex rounded-lg bg-purple-100 p-3 dark:bg-purple-900">
              <svg className="h-6 w-6 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold">QR Codes</h3>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Generate QR codes for any link. Perfect for print materials and sharing.
            </p>
          </div>
        </div>
      </main>
    </div>
  )
}

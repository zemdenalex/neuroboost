import Layout from '../../components/Layout'
export default function Home(){
  return (
    <div className="h-full overflow-y-auto bg-black text-zinc-100 font-mono">
      {/* Hero Section */}
      <section className="min-h-screen flex flex-col items-center justify-center px-6 py-20">
        <h1 className="text-5xl font-bold">NeuroBoost</h1>
        <h2 className="text-xl mt-2">The Organizer for Non-Organizers</h2>
        <p className="text-zinc-400 mt-4 max-w-xl text-center">Built for neurodivergent minds...</p>
        <div className="mt-6 flex gap-3">
          <a className="px-4 py-2 bg-zinc-800 rounded border border-zinc-700" href="#/calendar">Get Started</a>
          <a className="px-4 py-2 bg-zinc-800 rounded border border-zinc-700" href="#/login">Sign In</a>
        </div>
      </section>

      {/* Problem Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">Traditional planners don't work for everyone</h3>
        <div className="grid grid-cols-2 gap-6">
          <div className="bg-zinc-900 p-4 rounded border border-zinc-800">❌ The Problem</div>
          <div className="bg-zinc-900 p-4 rounded border border-zinc-800">✓ Our Approach</div>
        </div>
      </section>

      {/* Features Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">How NeuroBoost Works</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">Time-Blocking First</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">Pushy Reminders</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">Plan vs Actual</div>
          <div className="bg-zinc-900 p-3 rounded border border-zinc-800">Smart Learning</div>
        </div>
      </section>

      {/* Philosophy Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">Our Philosophy</h3>
        <p className="text-zinc-400 max-w-2xl">Most productivity tools are designed by organized people...</p>
      </section>

      {/* CTA Section */}
      <section className="px-6 py-16 border-t border-zinc-800">
        <h3 className="text-2xl font-semibold mb-4">Ready to try a different approach?</h3>
      </section>

      {/* Footer */}
      <footer className="px-6 py-6 border-t border-zinc-800">© 2025 NeuroBoost</footer>
    </div>
  )
}

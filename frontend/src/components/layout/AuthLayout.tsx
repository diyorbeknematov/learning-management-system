import { Award, BookOpen, GraduationCap, PlayCircle } from 'lucide-react'
import { Link, Outlet } from 'react-router'
import { Toaster } from '@/components/ui/sonner'

const points = [
  { icon: BookOpen, text: 'Courses from real instructors' },
  { icon: PlayCircle, text: 'Study at your own pace, on any device' },
  { icon: Award, text: 'A certificate you can share and check' },
]

const year = new Date().getFullYear()

/** The whole screen for login, register and the password pages: the brand beside the form. */
export function AuthLayout() {
  return (
    <>
      <div className="grid min-h-screen lg:grid-cols-2">
        <aside className="relative hidden flex-col overflow-hidden bg-gradient-to-br from-primary via-indigo-700 to-indigo-950 p-12 text-primary-foreground lg:flex">
          {/* soft shapes behind the text */}
          <div className="pointer-events-none absolute -right-24 -top-24 size-96 rounded-full bg-white/10" />
          <div className="pointer-events-none absolute -bottom-32 -left-20 size-[28rem] rounded-full bg-white/5" />
          <div className="pointer-events-none absolute right-16 bottom-40 size-40 rounded-full border-2 border-white/20" />

          <Link to="/" className="relative flex items-center gap-2 text-lg font-semibold">
            <span className="flex size-9 items-center justify-center rounded-lg bg-white/15">
              <GraduationCap className="size-5" />
            </span>
            Edura
          </Link>

          <div className="relative my-auto max-w-lg space-y-10 py-12">
            <h2 className="text-5xl font-bold leading-[1.1] tracking-tight">Learn something new, one lesson at a time.</h2>

            <ul className="space-y-4">
              {points.map(({ icon: Icon, text }) => (
                <li key={text} className="flex items-center gap-4 text-lg">
                  <span className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-white/15">
                    <Icon className="size-5" />
                  </span>
                  {text}
                </li>
              ))}
            </ul>
          </div>

          <p className="relative text-sm opacity-70">© {year} Edura</p>
        </aside>

        <main className="flex flex-col px-6 py-8 sm:px-12">
          <Link to="/" className="mb-8 flex items-center gap-2 font-semibold lg:hidden">
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <GraduationCap className="size-5" />
            </span>
            Edura
          </Link>

          <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center">
            <Outlet />
          </div>
        </main>
      </div>

      <Toaster />
    </>
  )
}

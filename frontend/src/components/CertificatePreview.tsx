import { Award } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'

/**
 * The certificate drawn as a page: a frame, the name of the student, the
 * course, the date, the instructor and a QR code that leads to the public page
 * where anybody can check it. The paper is always light, like a printed one.
 */
export function CertificatePreview({
  student,
  course,
  instructor,
  date,
  number,
  className,
}: {
  student?: string
  course?: string
  instructor?: string
  date?: string
  number?: string
  className?: string
}) {
  const link = number ? `${window.location.origin}/verify/${number}` : ''

  return (
    <div
      className={cn(
        'aspect-[1.45/1] w-full rounded-xl bg-gradient-to-br from-amber-100 via-amber-50 to-white p-2 text-slate-800 shadow-md ring-1 ring-amber-200',
        className,
      )}
      style={{ containerType: 'inline-size' }}
      role="img"
      aria-label={`Certificate of ${student} for ${course}`}
    >
      <div className="relative flex h-full flex-col items-center justify-between overflow-hidden rounded-lg border-2 border-amber-400/80 bg-white/70 px-[5cqw] py-[3.5cqw] text-center">
        <div className="pointer-events-none absolute -left-[6cqw] -top-[6cqw] size-[22cqw] rounded-full bg-amber-200/40" />
        <div className="pointer-events-none absolute -bottom-[8cqw] -right-[6cqw] size-[26cqw] rounded-full bg-indigo-200/30" />

        <div className="relative flex flex-col items-center gap-1">
          <span className="flex size-[9cqw] items-center justify-center rounded-full bg-amber-400 text-white shadow">
            <Award className="size-[5cqw]" />
          </span>
          <p className="text-[1.9cqw] font-semibold uppercase tracking-[0.3em] text-amber-700">Certificate of completion</p>
        </div>

        <div className="relative space-y-1">
          <p className="text-[2.3cqw] text-slate-500">This certifies that</p>
          <p className="font-serif text-[6.5cqw] font-semibold italic leading-tight text-indigo-900">{student}</p>
          <p className="text-[2.3cqw] text-slate-500">has successfully completed the course</p>
          <p className="line-clamp-2 text-[3.8cqw] font-semibold leading-snug">{course}</p>
        </div>

        <div className="relative flex w-full items-end justify-between gap-2 text-left text-[2cqw] text-slate-500">
          <div className="space-y-0.5">
            <p>
              <span className="font-medium text-slate-700">{formatDate(date)}</span>
            </p>
            {instructor && <p>Instructor: {instructor}</p>}
            <p className="font-mono">{number}</p>
          </div>
          {link && (
            <div className="w-[12cqw]">
              <QRCodeSVG value={link} size={256} className="h-auto w-full" level="M" bgColor="transparent" fgColor="#312e81" />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

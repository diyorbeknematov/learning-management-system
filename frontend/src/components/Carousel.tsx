import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Children, useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// how many cards are in view; the width of one card is the share left after the gaps (1.5rem)
const item = 'shrink-0 snap-start basis-full sm:basis-[calc(50%-0.75rem)] lg:basis-[calc(33.333%-1rem)] xl:basis-[calc(25%-1.125rem)]'

/**
 * One row of cards that is moved sideways with two arrows (or by swiping). Four
 * cards are in view on a wide screen, so a row with more of them keeps going.
 */
export function Carousel({ children, label }: { children: ReactNode; label: string }) {
  const track = useRef<HTMLDivElement>(null)
  const [canBack, setCanBack] = useState(false)
  const [canForward, setCanForward] = useState(false)

  const update = useCallback(() => {
    const el = track.current
    if (!el) return

    setCanBack(el.scrollLeft > 4)
    setCanForward(el.scrollLeft + el.clientWidth < el.scrollWidth - 4)
  }, [])

  useEffect(() => {
    update()

    const el = track.current
    const observer = new ResizeObserver(update)

    if (el) observer.observe(el)

    return () => observer.disconnect()
  }, [update, children])

  function move(direction: 1 | -1) {
    const el = track.current
    if (el) el.scrollBy({ left: direction * el.clientWidth, behavior: 'smooth' })
  }

  return (
    <div className="relative" role="group" aria-roledescription="carousel" aria-label={`${label}: courses`}>
      <div
        ref={track}
        onScroll={update}
        className="flex snap-x snap-mandatory gap-6 overflow-x-auto scroll-smooth pb-3 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {Children.map(children, (child) => (
          <div className={cn(item, 'flex')}>
            <div className="w-full">{child}</div>
          </div>
        ))}
      </div>

      {canBack && (
        <Button
          type="button"
          variant="outline"
          size="icon-lg"
          aria-label="Previous courses"
          className="absolute -left-4 top-[40%] z-10 rounded-full bg-card shadow-md"
          onClick={() => move(-1)}
        >
          <ChevronLeft />
        </Button>
      )}
      {canForward && (
        <Button
          type="button"
          variant="outline"
          size="icon-lg"
          aria-label="Next courses"
          className="absolute -right-4 top-[40%] z-10 rounded-full bg-card shadow-md"
          onClick={() => move(1)}
        >
          <ChevronRight />
        </Button>
      )}
    </div>
  )
}

import { isRouteErrorResponse, useRouteError } from 'react-router'
import { Button } from '@/components/ui/button'

/** What a person sees when a page fails to load or breaks: the cause, and a way to try again. */
export function RouteError() {
  const error = useRouteError()

  const message = isRouteErrorResponse(error) ? `${error.status} ${error.statusText}` : error instanceof Error ? error.message : 'Something went wrong.'

  return (
    <div className="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center gap-4 p-6 text-center">
      <p className="text-5xl font-bold text-primary">Oops</p>
      <h1 className="text-2xl font-semibold">This page could not be shown</h1>
      <p className="break-words rounded-lg bg-muted px-4 py-3 text-sm text-muted-foreground">{message}</p>
      <div className="flex gap-2">
        <Button onClick={() => window.location.reload()}>Reload the page</Button>
        <Button variant="outline" onClick={() => window.location.assign('/')}>
          Go to the home page
        </Button>
      </div>
    </div>
  )
}

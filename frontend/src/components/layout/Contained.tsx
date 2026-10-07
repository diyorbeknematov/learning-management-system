import { Outlet } from 'react-router'
import { container } from './container'

/** The frame of most pages: as wide as the screen allows, with room at the sides. */
export function Contained() {
  return (
    <div className={`${container} py-8`}>
      <Outlet />
    </div>
  )
}

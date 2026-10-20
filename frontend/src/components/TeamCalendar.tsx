import { useState } from 'react'
import type { CalendarEntry } from '../api'

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const MAX_VISIBLE = 4

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function isoDate(year: number, month: number, day: number) {
  return `${year}-${pad(month + 1)}-${pad(day)}`
}

export default function TeamCalendar({ entries }: { entries: CalendarEntry[] }) {
  const today = new Date()
  const [year, setYear] = useState(today.getFullYear())
  const [month, setMonth] = useState(today.getMonth())

  function shiftMonth(delta: number) {
    let m = month + delta
    let y = year
    if (m < 0) {
      m = 11
      y -= 1
    } else if (m > 11) {
      m = 0
      y += 1
    }
    setMonth(m)
    setYear(y)
  }

  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const firstWeekday = new Date(year, month, 1).getDay()
  const todayIso = isoDate(today.getFullYear(), today.getMonth(), today.getDate())

  const cells: (number | null)[] = Array(firstWeekday).fill(null)
  for (let d = 1; d <= daysInMonth; d++) cells.push(d)
  while (cells.length % 7 !== 0) cells.push(null)

  function entriesOn(dateIso: string) {
    return entries
      .filter((e) => e.start_date <= dateIso && e.end_date >= dateIso)
      .sort((a, b) => a.employee_name.localeCompare(b.employee_name))
  }

  return (
    <div className="calendar">
      <div className="calendar-header">
        <button type="button" className="calendar-nav-btn" onClick={() => shiftMonth(-1)} aria-label="Previous month">
          &#8249;
        </button>
        <span className="calendar-title">
          {new Date(year, month).toLocaleString('en-US', { month: 'long' })} {year}
        </span>
        <button type="button" className="calendar-nav-btn" onClick={() => shiftMonth(1)} aria-label="Next month">
          &#8250;
        </button>
      </div>
      <div className="calendar-grid">
        {WEEKDAYS.map((w) => (
          <div key={w} className="calendar-weekday">
            {w}
          </div>
        ))}
        {cells.map((day, i) => {
          if (day === null) return <div key={i} className="calendar-cell calendar-cell-empty" />
          const dateIso = isoDate(year, month, day)
          const dayEntries = entriesOn(dateIso)
          const isToday = dateIso === todayIso
          const visible = dayEntries.slice(0, MAX_VISIBLE)
          const overflow = dayEntries.length - visible.length
          return (
            <div key={i} className="calendar-cell">
              <span className={isToday ? 'calendar-day-number calendar-day-number-today' : 'calendar-day-number'}>
                {day}
              </span>
              <div className="calendar-entries">
                {visible.map((e, j) => (
                  <div
                    key={j}
                    className="calendar-entry"
                    title={`${e.employee_name}, ${e.leave_type_name}, ${e.start_date} to ${e.end_date}`}
                  >
                    <span className="calendar-entry-name">{e.employee_name.split(' ')[0]}</span>
                    <span className="calendar-entry-type">{e.leave_type_name}</span>
                  </div>
                ))}
                {overflow > 0 && <div className="calendar-entry-more">+{overflow} more</div>}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

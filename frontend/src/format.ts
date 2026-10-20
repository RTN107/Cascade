// Every amount shown anywhere in the app goes through this, so a payroll run's floating
// point division (basic pay divided by working days) never prints a long decimal tail.
export function money(amount: number) {
  return Math.round(amount * 10) / 10
}

// Payroll amounts are Indian Rupees: the rupee sign plus Indian digit grouping
// (lakhs and crores, e.g. 3,63,440), not the Western thousands grouping.
export function rupees(amount: number) {
  return `₹${money(amount).toLocaleString('en-IN')}`
}

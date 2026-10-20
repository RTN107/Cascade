package payroll

import "math"

// This is the one place the payroll formula lives. Everything else calls
// CalculateNetPay; change the math here to change the formula for every payslip.
// daysPerMonth is the example rate used to turn a day of Loss of Pay into a deduction
// amount: Basic Pay divided by daysPerMonth gives one day's pay. This is a working-days
// convention that varies by organization; 22 is this one's example value.
const daysPerMonth = 22.0

// basicPay is this employee's amount for the one component flagged is_base, the anchor
// the Loss of Pay daily rate is computed from. grossEarnings and totalDeductions are the
// sums of every earning and deduction component configured for this employee (Basic Pay
// is itself one of the earnings already counted in grossEarnings).
func CalculateNetPay(basicPay, grossEarnings, totalDeductions, lopDays float64) (lopDeduction, netPay float64) {
	dailyRate := basicPay / daysPerMonth
	lopDeduction = math.Floor(dailyRate * lopDays)
	netPay = grossEarnings - totalDeductions - lopDeduction
	return lopDeduction, netPay
}

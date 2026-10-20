import { jsPDF } from 'jspdf'
import type { ComponentLine, DeductionLine, Payslip } from './api'
import { money } from './format'

const INK = '#221E20'
const INK_MUTED = '#8a8381'
const ACCENT = '#73293A'
const RULE = '#d8d3cd'

// matches the en-IN grouping format.ts uses on screen, so a figure never prints
// differently in the PDF than it does in the app
function fmt(amount: number) {
  return money(amount).toLocaleString('en-IN')
}

function buildPayslipPdf(
  payslip: Payslip,
  employeeName: string,
  period: string,
  components: ComponentLine[],
  deductions: DeductionLine[],
) {
  const doc = new jsPDF({ unit: 'pt', format: 'a4' })
  const pageWidth = doc.internal.pageSize.getWidth()
  const marginX = 56
  let y = 72

  doc.setFont('helvetica', 'bold')
  doc.setFontSize(20)
  doc.setTextColor(INK)
  doc.text('Payslip', marginX, y)

  doc.setFont('helvetica', 'normal')
  doc.setFontSize(11)
  doc.setTextColor(INK_MUTED)
  doc.text(period, pageWidth - marginX, y, { align: 'right' })

  y += 28
  doc.setDrawColor(RULE)
  doc.line(marginX, y, pageWidth - marginX, y)

  y += 28
  doc.setFont('helvetica', 'normal')
  doc.setFontSize(10)
  doc.setTextColor(INK_MUTED)
  doc.text('Employee', marginX, y)
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(13)
  doc.setTextColor(INK)
  doc.text(employeeName, marginX, y + 18)

  y += 48

  function sectionHeading(label: string) {
    doc.setFont('helvetica', 'bold')
    doc.setFontSize(10)
    doc.setTextColor(ACCENT)
    doc.text(label.toUpperCase(), marginX, y)
    doc.setDrawColor(RULE)
    doc.line(marginX, y + 6, pageWidth - marginX, y + 6)
    y += 26
  }

  function row(label: string, amount: number, sub?: string) {
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(11)
    doc.setTextColor(INK)
    doc.text(label, marginX, y)
    doc.text(fmt(amount), pageWidth - marginX, y, { align: 'right' })
    y += 16
    if (sub) {
      doc.setFont('helvetica', 'normal')
      doc.setFontSize(9)
      doc.setTextColor(INK_MUTED)
      const lines = doc.splitTextToSize(sub, pageWidth - marginX * 2)
      doc.text(lines, marginX, y)
      y += lines.length * 12 + 4
    } else {
      y += 4
    }
  }

  const earnings = components.filter((c) => c.kind === 'earning')
  const otherDeductions = components.filter((c) => c.kind === 'deduction')

  sectionHeading('Earnings')
  if (earnings.length === 0) {
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(10)
    doc.setTextColor(INK_MUTED)
    doc.text('No earnings on record', marginX, y)
    y += 20
  } else {
    earnings.forEach((c) => row(c.name, c.amount))
  }

  y += 12
  sectionHeading('Deductions')
  if (otherDeductions.length === 0 && deductions.length === 0) {
    doc.setFont('helvetica', 'normal')
    doc.setFontSize(10)
    doc.setTextColor(INK_MUTED)
    doc.text('No deductions on record', marginX, y)
    y += 20
  } else {
    otherDeductions.forEach((c) => row(c.name, c.amount))
    deductions.forEach((d) => row('Loss of pay', d.amount, d.description))
  }

  y += 20
  doc.setDrawColor(RULE)
  doc.line(marginX, y, pageWidth - marginX, y)
  y += 24

  row('Gross earnings', payslip.gross_earnings)
  row('Total deductions', payslip.total_deductions)
  if (payslip.lop_deduction > 0) {
    row('Loss of pay deduction', payslip.lop_deduction)
  }

  y += 10
  doc.setFillColor(ACCENT)
  doc.roundedRect(marginX, y, pageWidth - marginX * 2, 44, 6, 6, 'F')
  doc.setFont('helvetica', 'bold')
  doc.setFontSize(12)
  doc.setTextColor('#F2F0EB')
  doc.text('Net pay', marginX + 16, y + 28)
  doc.setFontSize(16)
  doc.text(fmt(payslip.net_pay), pageWidth - marginX - 16, y + 29, { align: 'right' })

  return doc
}

function filenameFor(employeeName: string, period: string) {
  return `payslip-${employeeName.toLowerCase().replace(/\s+/g, '-')}-${period.toLowerCase().replace(/\s+/g, '-')}.pdf`
}

export function downloadPayslipPdf(
  payslip: Payslip,
  employeeName: string,
  period: string,
  components: ComponentLine[],
  deductions: DeductionLine[],
) {
  const doc = buildPayslipPdf(payslip, employeeName, period, components, deductions)
  doc.save(filenameFor(employeeName, period))
}

export function payslipPdfUrl(
  payslip: Payslip,
  employeeName: string,
  period: string,
  components: ComponentLine[],
  deductions: DeductionLine[],
) {
  const doc = buildPayslipPdf(payslip, employeeName, period, components, deductions)
  return doc.output('bloburl').toString()
}

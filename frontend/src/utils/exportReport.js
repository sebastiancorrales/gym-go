import jsPDF from 'jspdf';
import autoTable from 'jspdf-autotable';
import * as XLSX from 'xlsx';

/**
 * Export data as PDF table
 * @param {string} title - Report title
 * @param {string[]} headers - Column headers
 * @param {Array<Array>} rows - Table rows (arrays of values)
 * @param {string} filename - Output filename without extension
 */
export function exportPDF(title, headers, rows, filename) {
  const doc = new jsPDF();
  doc.setFontSize(16);
  doc.text(title, 14, 20);
  doc.setFontSize(9);
  doc.setTextColor(100);
  doc.text(`Generado: ${new Date().toLocaleString('es-CO')}`, 14, 28);

  autoTable(doc, {
    head: [headers],
    body: rows,
    startY: 34,
    styles: { fontSize: 9, cellPadding: 3 },
    headStyles: { fillColor: [16, 185, 129], textColor: 255, fontStyle: 'bold' },
    alternateRowStyles: { fillColor: [245, 245, 245] },
  });

  doc.save(`${filename}.pdf`);
}

/**
 * Export data as Excel file
 * @param {string} title - Sheet name
 * @param {string[]} headers - Column headers
 * @param {Array<Array>} rows - Table rows (arrays of values)
 * @param {string} filename - Output filename without extension
 */
export function exportExcel(title, headers, rows, filename) {
  const wsData = [headers, ...rows];
  const ws = XLSX.utils.aoa_to_sheet(wsData);

  // Auto-width columns
  const colWidths = headers.map((h, i) => {
    const maxLen = Math.max(h.length, ...rows.map(r => String(r[i] ?? '').length));
    return { wch: Math.min(maxLen + 2, 40) };
  });
  ws['!cols'] = colWidths;

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, title.slice(0, 31));
  XLSX.writeFile(wb, `${filename}.xlsx`);
}

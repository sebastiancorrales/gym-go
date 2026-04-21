import jsPDF from 'jspdf';
import autoTable from 'jspdf-autotable';
import * as XLSX from 'xlsx';

const fmt = v => new Intl.NumberFormat('es-CO', { style: 'currency', currency: 'COP', minimumFractionDigits: 0 }).format(v ?? 0);
const fmtDate = s => s ? new Date(s + 'T00:00:00').toLocaleDateString('es-CO', { day: '2-digit', month: 'short', year: 'numeric' }) : '';
const PM_LABEL = { EFECTIVO: 'Efectivo', TRANSFERENCIA: 'Transferencia', OTRO: 'Otro' };
const pmLabel = m => PM_LABEL[m?.toUpperCase?.()] ?? m ?? 'Otro';

/**
 * Exporta el reporte consolidado (Cuadre de Caja) en PDF
 */
export function exportConsolidadoPDF(data, dateRange, gymName = 'Gimnasio') {
  const doc   = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4' });
  const W     = doc.internal.pageSize.getWidth();
  const GRAY  = [100, 100, 100];
  const BLACK = [15, 15, 15];
  const GREEN = [5, 150, 105];
  const BLUE  = [37, 99, 235];
  const DARK  = [17, 24, 39];

  let y = 0;

  // ── Encabezado ──────────────────────────────────────────────────────────────
  doc.setFillColor(...DARK);
  doc.rect(0, 0, W, 38, 'F');

  doc.setTextColor(255, 255, 255);
  doc.setFontSize(18);
  doc.setFont('helvetica', 'bold');
  doc.text('CUADRE DE CAJA — REPORTE CONSOLIDADO', W / 2, 14, { align: 'center' });

  doc.setFontSize(10);
  doc.setFont('helvetica', 'normal');
  doc.text(gymName, W / 2, 22, { align: 'center' });

  doc.setFontSize(8.5);
  doc.setTextColor(180, 220, 180);
  doc.text(`Período: ${fmtDate(dateRange.s)} — ${fmtDate(dateRange.e)}`, W / 2, 29, { align: 'center' });
  doc.text(`Generado: ${new Date().toLocaleString('es-CO')}`, W / 2, 34.5, { align: 'center' });

  y = 46;

  // ── Resumen ejecutivo (3 cajas) ─────────────────────────────────────────────
  const boxW = (W - 28) / 3;
  const boxes = [
    { label: 'Suscripciones', value: fmt(data.subsRevenue),   sub: `${data.subs.length} registro${data.subs.length !== 1 ? 's' : ''}`, color: GREEN },
    { label: 'Ventas inventario', value: fmt(data.salesRevenue), sub: `${data.salesReport?.total_sales ?? data.salesList?.length ?? 0} transacción(es)`, color: BLUE },
    { label: 'TOTAL GENERAL',  value: fmt(data.grandTotal),   sub: 'Suscripciones + Ventas', color: DARK },
  ];

  boxes.forEach((b, i) => {
    const x = 14 + i * (boxW + 3);
    doc.setFillColor(...b.color);
    doc.roundedRect(x, y, boxW, 22, 2, 2, 'F');
    doc.setTextColor(255, 255, 255);
    doc.setFontSize(7);
    doc.setFont('helvetica', 'bold');
    doc.text(b.label.toUpperCase(), x + boxW / 2, y + 6, { align: 'center' });
    doc.setFontSize(12);
    doc.text(b.value, x + boxW / 2, y + 13.5, { align: 'center' });
    doc.setFontSize(7);
    doc.setFont('helvetica', 'normal');
    doc.text(b.sub, x + boxW / 2, y + 19.5, { align: 'center' });
  });

  y += 30;

  // ── Helper: título de sección ────────────────────────────────────────────────
  const sectionTitle = (title) => {
    doc.setFillColor(243, 244, 246);
    doc.rect(14, y, W - 28, 7, 'F');
    doc.setTextColor(...BLACK);
    doc.setFontSize(8.5);
    doc.setFont('helvetica', 'bold');
    doc.text(title, 17, y + 5);
    y += 10;
  };

  // ── Sección 1: Suscripciones ─────────────────────────────────────────────────
  sectionTitle('1. SUSCRIPCIONES DEL PERÍODO');

  if (data.subs.length === 0) {
    doc.setFontSize(8);
    doc.setFont('helvetica', 'italic');
    doc.setTextColor(...GRAY);
    doc.text('Sin suscripciones en este período', 17, y);
    y += 8;
  } else {
    const subRows = data.subs.map(s => {
      const nombre = `${s.user?.first_name || ''} ${s.user?.last_name || ''}`.trim();
      const grupo  = s.members?.length > 0
        ? s.members.filter(m => !m.is_primary).map(m => `${m.user?.first_name || ''} ${m.user?.last_name || ''}`.trim()).join(', ')
        : '';
      return [
        grupo ? `${nombre}\n+ ${grupo}` : nombre,
        s.plan?.name || '—',
        `${(s.start_date || '').split('T')[0]} → ${(s.end_date || '').split('T')[0]}`,
        pmLabel(s.payment_method),
        fmt(s.total_paid || 0),
      ];
    });

    autoTable(doc, {
      head: [['Usuario / Grupo', 'Plan', 'Período', 'Método', 'Total']],
      body: subRows,
      startY: y,
      margin: { left: 14, right: 14 },
      styles:          { fontSize: 7.5, cellPadding: 2.5, textColor: BLACK },
      headStyles:      { fillColor: GREEN, textColor: 255, fontStyle: 'bold', fontSize: 8 },
      columnStyles:    { 4: { halign: 'right', fontStyle: 'bold' } },
      alternateRowStyles: { fillColor: [240, 253, 244] },
      foot: [[{ content: 'Subtotal suscripciones', colSpan: 4, styles: { fontStyle: 'bold', halign: 'right', fillColor: [209, 250, 229], textColor: GREEN } },
              { content: fmt(data.subsRevenue), styles: { fontStyle: 'bold', halign: 'right', fillColor: [209, 250, 229], textColor: GREEN } }]],
      showFoot: 'lastPage',
      didDrawPage: (d) => { y = d.cursor.y + 6; },
    });
  }

  // ── Sección 2: Ventas inventario ─────────────────────────────────────────────
  if (y > 220) { doc.addPage(); y = 20; }
  sectionTitle('2. VENTAS DE INVENTARIO');

  const sr = data.salesReport;
  if (!sr && (!data.salesList || data.salesList.length === 0)) {
    doc.setFontSize(8);
    doc.setFont('helvetica', 'italic');
    doc.setTextColor(...GRAY);
    doc.text('Sin ventas de inventario en este período', 17, y);
    y += 8;
  } else {
    autoTable(doc, {
      head: [['Transacciones', 'Ingresos brutos', 'Descuentos', 'Ingresos netos']],
      body: [[
        sr?.total_sales ?? (data.salesList?.length ?? 0),
        fmt(sr?.gross_sales ?? 0),
        fmt(sr?.total_discounts ?? 0),
        fmt(data.salesRevenue),
      ]],
      startY: y,
      margin: { left: 14, right: 14 },
      styles:     { fontSize: 9, cellPadding: 3, halign: 'center', textColor: BLACK },
      headStyles: { fillColor: BLUE, textColor: 255, fontStyle: 'bold', fontSize: 8, halign: 'center' },
      columnStyles: { 3: { fontStyle: 'bold', textColor: BLUE } },
      didDrawPage: (d) => { y = d.cursor.y + 6; },
    });
  }

  // ── Sección 3: Desglose por método de pago ───────────────────────────────────
  if (y > 220) { doc.addPage(); y = 20; }
  sectionTitle('3. DESGLOSE POR MÉTODO DE PAGO');

  const methodRows = Object.entries(data.grandByMethod)
    .sort((a, b) => b[1].total - a[1].total)
    .map(([m, v]) => [
      pmLabel(m),
      v.subsCount  > 0 ? `${fmt(v.subsTotal)} (${v.subsCount})` : '—',
      v.salesCount > 0 ? `${fmt(v.salesTotal)} (${v.salesCount})` : '—',
      fmt(v.total),
    ]);

  autoTable(doc, {
    head: [['Método de pago', 'Suscripciones', 'Ventas inventario', 'Total']],
    body: methodRows,
    startY: y,
    margin: { left: 14, right: 14 },
    styles:       { fontSize: 8, cellPadding: 3, textColor: BLACK },
    headStyles:   { fillColor: [55, 65, 81], textColor: 255, fontStyle: 'bold', fontSize: 8 },
    columnStyles: { 3: { halign: 'right', fontStyle: 'bold' } },
    alternateRowStyles: { fillColor: [249, 250, 251] },
    didDrawPage: (d) => { y = d.cursor.y + 6; },
  });

  // ── Cuadre de Caja ───────────────────────────────────────────────────────────
  if (y > 240) { doc.addPage(); y = 20; }

  y += 4;
  // Línea de cierre
  doc.setDrawColor(...DARK);
  doc.setLineWidth(0.8);
  doc.line(14, y, W - 14, y);
  y += 6;

  doc.setFillColor(...DARK);
  doc.roundedRect(14, y, W - 28, 22, 3, 3, 'F');

  doc.setTextColor(150, 250, 200);
  doc.setFontSize(8);
  doc.setFont('helvetica', 'bold');
  doc.text('TOTAL GENERAL DEL PERÍODO', W / 2, y + 7, { align: 'center' });

  doc.setTextColor(255, 255, 255);
  doc.setFontSize(20);
  doc.text(fmt(data.grandTotal), W / 2, y + 17, { align: 'center' });

  y += 28;

  doc.setDrawColor(...DARK);
  doc.setLineWidth(0.8);
  doc.line(14, y, W - 14, y);

  y += 8;
  doc.setFontSize(7);
  doc.setFont('helvetica', 'italic');
  doc.setTextColor(...GRAY);
  doc.text(`Cuadre de caja generado el ${new Date().toLocaleString('es-CO')} — gym-go`, W / 2, y, { align: 'center' });

  // ── Numeración de páginas ────────────────────────────────────────────────────
  const pages = doc.internal.getNumberOfPages();
  for (let i = 1; i <= pages; i++) {
    doc.setPage(i);
    doc.setFontSize(7);
    doc.setTextColor(...GRAY);
    doc.text(`Página ${i} de ${pages}`, W - 14, doc.internal.pageSize.getHeight() - 8, { align: 'right' });
  }

  doc.save(`cuadre-caja_${dateRange.s}_${dateRange.e}.pdf`);
}

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

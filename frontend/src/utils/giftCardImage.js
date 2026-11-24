import qrcode from 'qrcode-generator';

// Dibujo de la tarjeta regalo en un <canvas> de 1080×1350 (formato vertical de
// WhatsApp/Instagram). Portado del MVP de Olimpo Gym: mismo diseño negro y
// dorado, pero los datos de contacto, el logo y la frase salen de la
// configuración del gimnasio (GET /gift-cards/config) en vez de estar fijos.
//
// El QR codifica sólo el código: lo lee un lector USB en la casilla de
// "Redimir", que se comporta como un teclado.

export const CARD_W = 1080;
export const CARD_H = 1350;

const MESES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio',
  'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];

/** "2026-11-12" → "12 de noviembre de 2026" */
export function fechaLarga(iso) {
  if (!iso) return '';
  const [y, m, d] = iso.split('-');
  return `${+d} de ${MESES[+m - 1]} de ${y}`;
}

/** Celular colombiano de 10 dígitos → con indicativo 57, como lo pide wa.me. */
export function waPhone(raw) {
  const d = String(raw || '').replace(/\D/g, '');
  if (d.length === 10 && d[0] === '3') return '57' + d;
  return d;
}

export function pesos(n) {
  return '$' + Math.round(n || 0).toLocaleString('es-CO');
}

export function whatsappText(card, config) {
  const gym = config?.gym_name || 'el gimnasio';
  return `Hola ${card.recipient_name}. ${card.buyer_name} te regaló ${String(card.plan_name).toLowerCase()} en ${gym}. `
    + `Tu código es ${card.code}. Actívalo en recepción antes del ${fechaLarga(card.expires_on)}. `
    + `${config?.address ? config.address + '. ' : ''}¡Te esperamos!`;
}

/** Carga el logo para dibujarlo. Resuelve null si no hay o no carga. */
export function loadLogo(url) {
  return new Promise((resolve) => {
    if (!url) { resolve(null); return; }
    const img = new Image();
    // Mismo origen (/uploads) en producción; sin esto un logo externo dejaría
    // el canvas "contaminado" y no se podría copiar ni descargar la imagen.
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = url;
  });
}

function wrap(ctx, text, width) {
  const words = String(text || '').split(/\s+/);
  const lines = [];
  let line = '';
  for (const w of words) {
    const test = line ? `${line} ${w}` : w;
    if (ctx.measureText(test).width > width && line) { lines.push(line); line = w; } else line = test;
  }
  if (line) lines.push(line);
  return lines;
}

function drawQR(ctx, text, x, y, size) {
  const qr = qrcode(0, 'M');
  qr.addData(text);
  qr.make();
  const n = qr.getModuleCount();
  const margin = 2;
  const cell = size / (n + margin * 2);
  ctx.fillStyle = '#f2ebdd';
  ctx.fillRect(x, y, size, size);
  ctx.fillStyle = '#141210';
  for (let r = 0; r < n; r++) {
    for (let c = 0; c < n; c++) {
      if (qr.isDark(r, c)) {
        ctx.fillRect(Math.floor(x + (c + margin) * cell), Math.floor(y + (r + margin) * cell),
          Math.ceil(cell), Math.ceil(cell));
      }
    }
  }
}

export function drawGiftCard(canvas, card, config, logo) {
  const ctx = canvas.getContext('2d');
  const W = CARD_W;
  const H = CARD_H;
  canvas.width = W;
  canvas.height = H;
  const oro = '#c9a24a';
  const oroOsc = '#6e5626';
  const crema = '#f2ebdd';
  const suave = '#c9bea6';
  const serif = 'Georgia, "Times New Roman", serif';
  const sans = '"Segoe UI", Arial, sans-serif';

  ctx.fillStyle = '#141210';
  ctx.fillRect(0, 0, W, H);
  ctx.strokeStyle = oro; ctx.lineWidth = 3; ctx.strokeRect(36, 36, W - 72, H - 72);
  ctx.strokeStyle = oroOsc; ctx.lineWidth = 1.5; ctx.strokeRect(52, 52, W - 104, H - 104);
  ctx.textAlign = 'center';

  ctx.fillStyle = oro; ctx.font = `600 26px ${sans}`;
  ctx.fillText('T A R J E T A   D E   R E G A L O', W / 2, 122);

  // Logo (o las iniciales del gimnasio si no tiene).
  const r = 150; const cx = W / 2; const cy = 300;
  ctx.save();
  ctx.beginPath(); ctx.arc(cx, cy, r, 0, Math.PI * 2); ctx.closePath(); ctx.clip();
  if (logo) {
    ctx.fillStyle = '#ffffff'; ctx.fillRect(cx - r, cy - r, r * 2, r * 2);
    ctx.drawImage(logo, cx - r, cy - r, r * 2, r * 2);
  } else {
    ctx.fillStyle = '#1f1b16'; ctx.fillRect(cx - r, cy - r, r * 2, r * 2);
    const initials = String(config?.gym_name || 'GYM').split(/\s+/).map(w => w[0]).join('').slice(0, 3).toUpperCase();
    ctx.fillStyle = oro; ctx.font = `700 110px ${serif}`; ctx.textBaseline = 'middle';
    ctx.fillText(initials, cx, cy + 6);
    ctx.textBaseline = 'alphabetic';
  }
  ctx.restore();
  ctx.beginPath(); ctx.arc(cx, cy, r, 0, Math.PI * 2); ctx.strokeStyle = oro; ctx.lineWidth = 4; ctx.stroke();

  // Plan
  ctx.fillStyle = crema; ctx.font = `700 58px ${serif}`;
  let y = 530;
  for (const l of wrap(ctx, card.plan_name, 880)) { ctx.fillText(l, W / 2, y); y += 66; }

  // Para / De
  y += 24;
  ctx.fillStyle = oro; ctx.font = `600 26px ${sans}`; ctx.fillText('PARA', W / 2, y);
  ctx.fillStyle = crema; ctx.font = `700 50px ${serif}`; ctx.fillText(card.recipient_name, W / 2, y + 58);
  y += 112;
  ctx.fillStyle = suave; ctx.font = `400 30px ${sans}`; ctx.fillText(`De parte de ${card.buyer_name}`, W / 2, y);
  if (card.dedication) {
    ctx.fillStyle = suave; ctx.font = `italic 28px ${serif}`;
    y += 46;
    for (const l of wrap(ctx, `“${card.dedication}”`, 820).slice(0, 2)) { ctx.fillText(l, W / 2, y); y += 38; }
  }

  // Código y QR
  const yb = 935;
  ctx.strokeStyle = oroOsc; ctx.lineWidth = 1.5;
  ctx.beginPath(); ctx.moveTo(120, yb - 40); ctx.lineTo(W - 120, yb - 40); ctx.stroke();
  drawQR(ctx, card.code, 140, yb, 190);
  ctx.textAlign = 'left';
  ctx.fillStyle = oro; ctx.font = `600 24px ${sans}`; ctx.fillText('CÓDIGO', 370, yb + 36);
  ctx.fillStyle = crema; ctx.font = `700 64px ${serif}`; ctx.fillText(card.code, 370, yb + 104);
  ctx.fillStyle = suave; ctx.font = `400 24px ${sans}`;
  ctx.fillText(`Actívala antes del ${fechaLarga(card.expires_on)}`, 370, yb + 148);
  ctx.fillText('Preséntala en recepción', 370, yb + 182);

  // Frase y contacto
  ctx.textAlign = 'center';
  ctx.fillStyle = crema; ctx.font = `italic 30px ${serif}`;
  ctx.fillText(config?.settings?.phrase || '', W / 2, 1200);
  ctx.fillStyle = suave; ctx.font = `400 22px ${sans}`;
  const contact = [config?.phone && `WhatsApp ${config.phone}`, config?.instagram].filter(Boolean).join('   ');
  if (contact) ctx.fillText(contact, W / 2, 1240);
  if (config?.address) ctx.fillText(config.address, W / 2, 1272);
}

export function canvasToBlob(canvas) {
  return new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
}

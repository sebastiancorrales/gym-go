"""
parse_chat.py  v3
Parsea el chat de WhatsApp, cruza con la BD y genera UPDATEs de created_at.

Uso:
    python parse_chat.py          # dry-run, genera updates.sql / no_match.txt / report.txt
    python parse_chat.py --apply  # aplica directamente en la BD
"""

import re, sys, sqlite3
from datetime import datetime
from pathlib import Path

BASE_DIR  = Path(__file__).parent
CHAT_FILE = BASE_DIR / "WhatsApp Chat - Registros Olimpo" / "_chat.txt"
DB_FILE   = BASE_DIR / "gym-go.db"
APPLY     = "--apply" in sys.argv

# ────────────────────────────────────────────────────────────────────────────
# 1. LIMPIEZA DE CARACTERES INVISIBLES DE WHATSAPP
# ────────────────────────────────────────────────────────────────────────────
INVISIBLE = re.compile(r"[\u200e\u200f\u202a-\u202e\u202f\ufeff]")

def clean(s: str) -> str:
    """Elimina caracteres invisibles y normaliza espacios."""
    return INVISIBLE.sub(" ", s).strip()

# ────────────────────────────────────────────────────────────────────────────
# 2. PARSEO DE MENSAJES
# ────────────────────────────────────────────────────────────────────────────
MSG_RE = re.compile(
    r"^\[(\d{1,2}/\d{1,2}/\d{2,4},\s*\d{1,2}:\d{2}:\d{2}\s*[ap]\.m\.)\]\s+([^:]+):\s*(.*)"
)

def parse_date(s: str):
    s = s.replace("a.m.", "AM").replace("p.m.", "PM")
    for fmt in ("%d/%m/%y, %I:%M:%S %p", "%d/%m/%Y, %I:%M:%S %p"):
        try:
            return datetime.strptime(s, fmt)
        except ValueError:
            pass
    return None

# ────────────────────────────────────────────────────────────────────────────
# 3. DETECCIÓN DE CÉDULAS
#    Reglas:
#    - Entre 4 y 10 dígitos
#    - NO empieza con 3 + tiene 10 dígitos (teléfono colombiano)
#    - NO es monto: múltiplo de 500 >= 5000 (70000, 130000, 40000…)
#    - NO es monto con separador: 70.000, 130.000
# ────────────────────────────────────────────────────────────────────────────
PHONE_RE  = re.compile(r"^3\d{9}$")
MONTO_SEP = re.compile(r"^\d{1,3}([.,]\d{3})+$")   # 70.000 / 130,000

def is_cedula(token: str) -> bool:
    t = re.sub(r"[.\s]", "", token)   # quitar puntos y espacios
    if not re.match(r"^\d{4,10}$", t):
        return False
    if PHONE_RE.match(t):
        return False
    if MONTO_SEP.match(token.strip()):  # formato 70.000
        return False
    n = int(t)
    # Monto sin separador: múltiplo de 500 y >= 5000
    if n >= 5000 and n % 500 == 0:
        return False
    # Montos de clases / días sueltos comunes (1000–4500)
    if n in (1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500):
        return False
    return True

def extract_cedulas(lines: list) -> list:
    cedulas = []
    for line in lines:
        # Quitar prefijos tipo "CC:", "TI:", "C.C.", "N.T.", "Tel:"
        line = re.sub(
            r"^(CC|TI|C\.C\.|N\.T\.|Cc|cc|Tel|TEL|T\.E\.L|C\.C)[:\s#.]*",
            "", line
        ).strip()
        # Separar tokens por espacio, coma, punto y coma
        for token in re.split(r"[\s,;/]+", line):
            token = token.strip(".()")
            if is_cedula(token):
                raw = re.sub(r"[.\s]", "", token)
                cedulas.append(raw)
    return list(dict.fromkeys(cedulas))  # dedup preservando orden

# ────────────────────────────────────────────────────────────────────────────
# 4. FILTRO DE RUIDO
#    Solo filtramos líneas que CLARAMENTE no tienen cédulas ni planes.
#    Regla: si la línea contiene solo texto de operación (agua, clase, etc.)
#    NO la filtramos si podría ser un nombre de persona o contener un número.
# ────────────────────────────────────────────────────────────────────────────
NOISE_CONTAINS = [
    "imagen omitida", "se eliminó este mensaje", "eliminaste este mensaje",
    "los mensajes y las llamadas", "creaste el grupo", "añadiste a",
]
NOISE_STARTSWITH = [
    "agua ", "una agua", "dos agua",
    "gatorade", "gatorlit", "gatorlot", "gatorlite",
    "preentreno", "pre entreno", "pre-entreno",
    "proteína", "proteina", "creatina", "postobon",
    "camiseta", "esqueleto", "suplemento",
    "una clase", "dos clase", "tres clase", "2 clases", "3 clases", "4 clases",
    "un día", "dos días", "dos dias", "un dia",
    "caja ", "caja\n", "base ", "descuadre", "quedan ", "quedan\n",
    "final ", "total ", "cuadre ", "cierre", "ingreso",
    "corte ", "recibo ",
    "sandía", "sandia",
    "postobon",
]

def is_noise(line: str) -> bool:
    l = line.lower().strip()
    if not l:
        return True
    for kw in NOISE_CONTAINS:
        if kw in l:
            return True
    for kw in NOISE_STARTSWITH:
        if l.startswith(kw):
            return True
    # Línea que es SOLO un monto con separador de miles: 70.000 / 130.000
    if MONTO_SEP.match(l.replace(" ", "")):
        return True
    # Línea de monto sin separador seguido opcionalmente de método de pago
    if re.match(r"^\d{4,7}\s*(efectivo|nequi|qr|bbva|daviplata|transferencia|nequi)?$", l):
        return True
    return False

# ────────────────────────────────────────────────────────────────────────────
# 5. DETECCIÓN DE PLAN
# ────────────────────────────────────────────────────────────────────────────
PLAN_MAP = {
    "cuarteto": ["cuarteto"],
    "trio":     ["trio", "trío"],
    "panas":    ["panas", "amigos"],
    "pareja":   ["pareja", "parejas"],
    "mensual":  ["mensualidad","mensual","mesualidad","mensulidad",
                 "mesialidad","mesulidad","membresia","membresía"],
    "quincena": ["quincena", "quincenas"],
    "semana":   ["semana"],
}

def detect_plan(lines: list) -> str:
    text = " ".join(lines).lower()
    for key in ["cuarteto", "trio", "panas", "pareja", "quincena", "semana", "mensual"]:
        for kw in PLAN_MAP[key]:
            if kw in text:
                return key
    return None

# ────────────────────────────────────────────────────────────────────────────
# 6. PARSEO DEL CHAT COMPLETO
# ────────────────────────────────────────────────────────────────────────────
def parse_chat(path: Path) -> list:
    with open(path, encoding="utf-8") as f:
        raw_lines = f.readlines()

    # Agrupar líneas en mensajes
    messages = []
    current  = None
    for raw_line in raw_lines:
        line = clean(raw_line)
        m = MSG_RE.match(line)
        if m:
            if current:
                messages.append(current)
            current = {
                "date_str": m.group(1).strip(),
                "author":   m.group(2).strip(),
                "lines":    [m.group(3).strip()],
            }
        elif current and line:
            current["lines"].append(line)
    if current:
        messages.append(current)

    registros = []
    for msg in messages:
        dt = parse_date(msg["date_str"])
        if not dt:
            continue

        # Filtrar ruido SOLO en líneas de texto puro (no toca líneas numéricas)
        clean_lines = [l for l in msg["lines"] if l and not is_noise(l)]
        if not clean_lines:
            continue

        cedulas = extract_cedulas(clean_lines)
        if not cedulas:
            continue

        registros.append({
            "date":     dt,
            "cedulas":  cedulas,
            "plan_key": detect_plan(msg["lines"]),
            "raw":      " | ".join(msg["lines"][:10]),
        })

    return registros

# ────────────────────────────────────────────────────────────────────────────
# 7. CRUCE CON BD
# ────────────────────────────────────────────────────────────────────────────
def cross_with_db(registros: list, db_path: Path):
    conn = sqlite3.connect(str(db_path))
    conn.row_factory = sqlite3.Row
    cur  = conn.cursor()

    cur.execute("SELECT id, first_name, last_name, document_number FROM users WHERE document_number != ''")
    users_by_doc = {r["document_number"]: dict(r) for r in cur.fetchall()}

    updates  = []
    no_match = []

    for reg in registros:
        date_iso = reg["date"].strftime("%Y-%m-%dT%H:%M:%S")
        matched  = []
        missing  = []

        for doc in reg["cedulas"]:
            user = users_by_doc.get(doc)
            if not user:
                missing.append(f"{doc} (no en BD)")
                continue

            cur.execute("""
                SELECT s.id, s.created_at, p.name as plan_name
                FROM subscriptions s
                JOIN plans p ON s.plan_id = p.id
                WHERE s.user_id = ?
                ORDER BY s.created_at ASC
                LIMIT 1
            """, (user["id"],))
            sub = cur.fetchone()

            if sub:
                matched.append({
                    "sub_id":       sub["id"],
                    "user_name":    f"{user['first_name']} {user['last_name']}".strip(),
                    "doc":          doc,
                    "plan_name":    sub["plan_name"],
                    "current_date": sub["created_at"],
                    "new_date":     date_iso,
                })
            else:
                missing.append(f"{doc} (sin suscripcion)")

        if matched:
            updates.extend(matched)
        if missing or not matched:
            no_match.append({
                "date":    date_iso,
                "plan":    reg["plan_key"],
                "docs":    reg["cedulas"],
                "missing": missing,
                "raw":     reg["raw"],
            })

    conn.close()
    return updates, no_match

# ────────────────────────────────────────────────────────────────────────────
# 8. GENERAR SQL
# ────────────────────────────────────────────────────────────────────────────
def generate_sql(updates: list) -> str:
    lines = [
        "-- AUTO-GENERADO por parse_chat.py v3",
        "-- Revisar cada UPDATE antes de ejecutar en produccion.",
        "-- Actualiza created_at de subscriptions segun fecha del chat de WhatsApp.",
        "",
        "BEGIN TRANSACTION;",
        "",
    ]
    for u in updates:
        lines.append(f"-- {u['user_name']} | doc: {u['doc']} | plan: {u['plan_name']}")
        lines.append(f"-- Antes: {u['current_date']}  ->  Despues: {u['new_date']}")
        lines.append(
            f"UPDATE subscriptions SET created_at = '{u['new_date']}' WHERE id = '{u['sub_id']}';"
        )
        lines.append("")
    lines += ["COMMIT;", ""]
    return "\n".join(lines)

# ────────────────────────────────────────────────────────────────────────────
# 9. MAIN
# ────────────────────────────────────────────────────────────────────────────
def main():
    print(f"Leyendo: {CHAT_FILE}")
    registros = parse_chat(CHAT_FILE)
    print(f"  Bloques con cedulas encontrados: {len(registros)}")

    print(f"Cruzando con BD: {DB_FILE}")
    updates, no_match = cross_with_db(registros, DB_FILE)

    # filtrar no_match que son pure "no en BD" (no son errores del script)
    real_no_match = [nm for nm in no_match if nm["missing"]]

    print(f"  Suscripciones a actualizar: {len(updates)}")
    print(f"  Cedulas no encontradas en BD: {len(real_no_match)}")

    # Guardar archivos
    (BASE_DIR / "updates.sql").write_text(generate_sql(updates), encoding="utf-8")

    nm_lines = []
    for nm in real_no_match:
        nm_lines.append(f"[{nm['date']}] plan={nm['plan']} docs={nm['docs']}")
        nm_lines.append(f"  Faltantes: {nm['missing']}")
        nm_lines.append(f"  Raw: {nm['raw'][:150]}")
        nm_lines.append("")
    (BASE_DIR / "no_match.txt").write_text("\n".join(nm_lines), encoding="utf-8")

    report = [
        "=" * 72,
        f"REPORTE - {datetime.now().strftime('%Y-%m-%d %H:%M')}",
        "=" * 72,
        f"Total bloques con cedulas: {len(registros)}",
        f"Suscripciones a actualizar: {len(updates)}",
        f"Cedulas no en BD: {len(real_no_match)} bloques",
        "",
        "-- ACTUALIZACIONES --",
    ]
    for u in updates:
        report.append(
            f"  {u['user_name']:<32} doc={u['doc']:<12} "
            f"plan={u['plan_name']:<22} "
            f"{u['current_date']} -> {u['new_date']}"
        )
    report += ["", "-- CEDULAS NO ENCONTRADAS EN BD --"]
    for nm in real_no_match:
        report.append(f"  [{nm['date']}] faltantes={nm['missing']}")
        report.append(f"    {nm['raw'][:100]}")

    (BASE_DIR / "report.txt").write_text("\n".join(report), encoding="utf-8")

    print(f"\nArchivos:")
    print(f"  updates.sql  -> {len(updates)} UPDATEs")
    print(f"  no_match.txt -> {len(real_no_match)} bloques sin resolver")
    print(f"  report.txt")

    if APPLY:
        print("\nAplicando UPDATEs en BD...")
        conn = sqlite3.connect(str(DB_FILE))
        try:
            cur = conn.cursor()
            cur.execute("BEGIN")
            count = 0
            for u in updates:
                cur.execute(
                    "UPDATE subscriptions SET created_at = ? WHERE id = ?",
                    (u["new_date"], u["sub_id"])
                )
                count += cur.rowcount
            conn.commit()
            print(f"  OK: {count} filas actualizadas")
        except Exception as e:
            conn.rollback()
            print(f"  ERROR: {e} - rollback aplicado, nada fue modificado")
        finally:
            conn.close()
    else:
        print("\nModo dry-run. Para aplicar: python parse_chat.py --apply")

if __name__ == "__main__":
    main()

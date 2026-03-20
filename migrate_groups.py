"""
Migra las suscripciones grupales existentes al nuevo modelo subscription_members.

Lo que hace:
1. Encuentra todas las suscripciones de planes grupales (PAREJA, PANAS, TRIO, CUARTETO)
2. Agrupa las que comparten plan_id + start_date (fueron creadas juntas)
3. La de total_paid > 0 es el titular; las de $0 son adicionales
4. Crea registros en subscription_members para cada grupo
5. Elimina las suscripciones adicionales ($0) — quedan como miembros del titular

Uso: python migrate_groups.py [--dry-run]
"""
import sys, sqlite3, uuid
from datetime import datetime

DB_PATH = r"c:\Users\sebas\OneDrive\Desktop\gym-go\gym-go.db"
DRY_RUN = "--dry-run" in sys.argv

con = sqlite3.connect(DB_PATH)
con.row_factory = sqlite3.Row
cur = con.cursor()

# Verificar que la tabla existe (la crea el AutoMigrate al iniciar el servidor)
cur.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='subscription_members'")
if not cur.fetchone():
    print("ERROR: La tabla subscription_members no existe.")
    print("Inicia el servidor una vez para que AutoMigrate la cree, luego ejecuta este script.")
    sys.exit(1)

# Obtener plan IDs de planes grupales
cur.execute("SELECT id, name FROM plans WHERE UPPER(name) IN ('PAREJA','PANAS','TRIO','CUARTETO')")
group_plans = {row["id"]: row["name"] for row in cur.fetchall()}

if not group_plans:
    print("No se encontraron planes grupales. Verifica que los planes existan.")
    sys.exit(0)

print(f"Planes grupales encontrados: {list(group_plans.values())}")
print()

# Obtener todas las suscripciones de planes grupales
placeholders = ",".join("?" * len(group_plans))
cur.execute(f"""
    SELECT s.id, s.user_id, s.plan_id, s.start_date, s.total_paid, s.created_at, p.name as plan_name
    FROM subscriptions s
    JOIN plans p ON s.plan_id = p.id
    WHERE s.plan_id IN ({placeholders})
    ORDER BY s.plan_id, s.start_date, s.total_paid DESC
""", list(group_plans.keys()))

subs = [dict(row) for row in cur.fetchall()]
print(f"Suscripciones grupales encontradas: {len(subs)}")

# Agrupar por plan_id + start_date (mismo día)
from collections import defaultdict
groups = defaultdict(list)
for s in subs:
    start = s["start_date"][:10] if s["start_date"] else "?"
    key = (s["plan_id"], start)
    groups[key].append(s)

print(f"Grupos identificados: {len(groups)}")
print()

stats = {"grupos": 0, "members_creados": 0, "subs_eliminadas": 0, "ya_migrados": 0}

for (plan_id, start_date), group_subs in groups.items():
    if len(group_subs) < 2:
        continue  # suscripción individual, no necesita migración

    # Verificar si ya tiene members registrados
    primary_id = next((s["id"] for s in group_subs if s["total_paid"] > 0), group_subs[0]["id"])
    cur.execute("SELECT COUNT(*) as c FROM subscription_members WHERE subscription_id = ?", (primary_id,))
    if cur.fetchone()["c"] > 0:
        stats["ya_migrados"] += 1
        continue

    plan_name = group_subs[0]["plan_name"]
    primary = next((s for s in group_subs if s["total_paid"] > 0), group_subs[0])
    additionals = [s for s in group_subs if s["id"] != primary["id"]]

    print(f"  {plan_name} | {start_date} | titular: {primary['user_id'][:8]}... | adicionales: {len(additionals)}")

    ahora = datetime.now().isoformat()

    if not DRY_RUN:
        # Crear member record para el titular
        cur.execute("""
            INSERT OR IGNORE INTO subscription_members (id, subscription_id, user_id, is_primary, created_at)
            VALUES (?, ?, ?, 1, ?)
        """, (str(uuid.uuid4()), primary["id"], primary["user_id"], ahora))
        stats["members_creados"] += 1

        # Crear member records para adicionales y eliminar sus suscripciones
        for s in additionals:
            cur.execute("""
                INSERT OR IGNORE INTO subscription_members (id, subscription_id, user_id, is_primary, created_at)
                VALUES (?, ?, ?, 0, ?)
            """, (str(uuid.uuid4()), primary["id"], s["user_id"], ahora))
            stats["members_creados"] += 1

            # Eliminar la suscripción adicional ($0)
            cur.execute("DELETE FROM subscriptions WHERE id = ?", (s["id"],))
            stats["subs_eliminadas"] += 1

    stats["grupos"] += 1

if not DRY_RUN:
    con.commit()
    print()
    print("Cambios guardados.")
else:
    print()
    print("[DRY RUN] No se guardaron cambios.")

con.close()

print()
print("=" * 50)
print("RESUMEN")
print("=" * 50)
print(f"  Grupos migrados:          {stats['grupos']}")
print(f"  Members creados:          {stats['members_creados']}")
print(f"  Suscripciones eliminadas: {stats['subs_eliminadas']}")
print(f"  Ya migrados (skip):       {stats['ya_migrados']}")

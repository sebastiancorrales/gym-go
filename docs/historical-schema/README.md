# Esquema histórico (no se ejecuta)

Estos `.sql` documentan la intención original del esquema, pero **ningún código Go
los lee ni los ejecuta**. El esquema real lo genera `db.AutoMigrate(...)` en
[`internal/infrastructure/persistence/migrations/migrate.go`](../../internal/infrastructure/persistence/migrations/migrate.go)
a partir de los structs de `internal/domain/entities/`.

Se movieron aquí porque estando en `migrations/` daban a entender que había un
sistema de migraciones SQL, y el instalador los copiaba a la máquina destino sin
que nadie los usara. Los índices que declaraban ya están recogidos como tags
`gorm:"index"` en las entidades.

Si en el futuro se adopta un sistema de migraciones de verdad, son un buen punto
de partida — pero hoy son documentación, no código.

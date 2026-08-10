// Command migcheck corre las migraciones contra una copia de una base real y
// mide cuánto tardan.
//
// Existe porque backfillPaymentLines no es reanudable y su error es fatal: si
// tarda minutos, el gimnasio no abre durante ese rato. Medirlo contra los datos
// de verdad antes de desplegar es la diferencia entre saberlo y descubrirlo.
//
// Uso: go run ./scripts/migcheck <ruta-a-una-copia.db>
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sebastiancorrales/gym-go/internal/config"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/persistence/migrations"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("uso: go run ./scripts/migcheck <ruta-a-una-copia.db>")
		os.Exit(2)
	}

	db, err := config.NewDatabase(&config.DatabaseConfig{DatabasePath: os.Args[1], ForMigration: true})
	if err != nil {
		fmt.Println("ERROR abriendo la base:", err)
		os.Exit(1)
	}
	defer db.Close()

	start := time.Now()
	if err := migrations.Migrate(db.DB); err != nil {
		fmt.Println("MIGRACION FALLO:", err)
		os.Exit(1)
	}
	fmt.Printf("\nMigrate() completo en %s\n", time.Since(start).Round(time.Millisecond))
}

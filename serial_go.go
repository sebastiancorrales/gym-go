package main

import (
	"fmt"

	"go.bug.st/serial"
)

func main() {
	mode := &serial.Mode{
		BaudRate: 9600,
	}

	// Cambia COM5 si es necesario
	port, err := serial.Open("COM5", mode)
	if err != nil {
		fmt.Println("Error abriendo puerto:", err)
		return
	}
	defer port.Close()

	// Espera para que Arduino reinicie
	// time.Sleep(2 * time.Second)

	// Enviar comando
	_, err = port.Write([]byte("OPEN\n"))
	if err != nil {
		fmt.Println("Error enviando comando:", err)
		return
	}

	fmt.Println("Comando enviado: OPEN")
}

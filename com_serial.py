import serial
import time

# Cambia COM5 si es necesario
port = "COM5"
baudrate = 9600

ser = serial.Serial(port, baudrate, timeout=1)

# time.sleep(2)  # Esperar a que el Arduino reinicie

# Enviar comando
ser.write(b"OPEN\n")

print("Comando enviado: OPEN")

ser.close()
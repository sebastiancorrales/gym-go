# Servicio de captura biométrica usando Windows Biometric Framework
# Este servicio se comunica con el lector U.are.U 4500 via WBF (nativo de Windows)
# No requiere SDK adicional, solo PowerShell

param(
    [int]$Port = 9000
)

Write-Host "Servicio Biométrico WBF - Puerto $Port" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Variables globales para almacenar templates capturados
$script:capturedTemplate = $null
$script:captureQuality = 0

function Start-BiometricListener {
    param([int]$Port)
    
    try {
        # Crear listener TCP
        $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
        $listener.Start()
        
        Write-Host "[OK] Servicio iniciado en localhost:$Port" -ForegroundColor Green
        Write-Host "[INFO] Esperando conexiones de la aplicación Go..." -ForegroundColor Yellow
        Write-Host ""
        
        while ($true) {
            # Aceptar conexión
            $client = $listener.AcceptTcpClient()
            Write-Host "[CONN] Nueva conexión recibida" -ForegroundColor Cyan
            
            try {
                $stream = $client.GetStream()
                $reader = [System.IO.StreamReader]::new($stream)
                $writer = [System.IO.StreamWriter]::new($stream)
                $writer.AutoFlush = $true
                
                # Leer comando
                $request = $reader.ReadLine()
                Write-Host "[REQ] $request" -ForegroundColor White
                
                # Parsear JSON
                $command = $request | ConvertFrom-Json
                
                # Procesar comando
                $response = switch ($command.command) {
                    "status" { Get-ReaderStatus }
                    "capture" { Invoke-CaptureFingerprint }
                    "verify" { Invoke-VerifyFingerprint -TemplateData $command.template_data }
                    default { @{ success = $false; message = "Unknown command: $($command.command)" } }
                }
                
                # Enviar respuesta
                $jsonResponse = $response | ConvertTo-Json -Compress
                $writer.WriteLine($jsonResponse)
                
                Write-Host "[RESP] success=$($response.success), message=$($response.message)" -ForegroundColor Green
                Write-Host ""
                
            } catch {
                Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
            } finally {
                if ($writer) { $writer.Close() }
                if ($reader) { $reader.Close() }
                if ($stream) { $stream.Close() }
                if ($client) { $client.Close() }
            }
        }
    } catch {
        Write-Host "[FATAL] Error al iniciar servicio: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    } finally {
        if ($listener) { $listener.Stop() }
    }
}

function Get-ReaderStatus {
    Write-Host "[CHECK] Verificando estado del lector..." -ForegroundColor Yellow
    
    # Verificar dispositivo WBF
    $device = Get-PnpDevice | Where-Object {
        $_.FriendlyName -like "*Fingerprint*" -and $_.Status -eq "OK"
    }
    
    if ($device) {
        return @{
            success = $true
            message = "Reader connected"
            data = @{
                connected = $true
                model = $device.FriendlyName
                status = $device.Status
            }
        }
    } else {
        return @{
            success = $false
            message = "No fingerprint reader found"
            data = @{
                connected = $false
            }
        }
    }
}

function Invoke-CaptureFingerprint {
    Write-Host "[CAPTURE] Iniciando captura de huella..." -ForegroundColor Yellow
    Write-Host "[INFO] Coloca tu dedo en el lector..." -ForegroundColor Cyan
    
    # Nota: Windows Biometric Framework requiere APIs nativas de .NET
    # Para una implementación completa, necesitarías usar C# o P/Invoke
    # Esta es una versión simplificada que simula la captura
    
    # En producción, aquí usarías:
    # - Windows.Devices.Input.Preview.BarcodeScanner (UWP)
    # - O P/Invoke a winbio.dll
    
    # Por ahora, simular captura para demostración
    Write-Host "[WARN] Captura simulada - requiere implementación con C# o P/Invoke" -ForegroundColor Yellow
    
    # Generar un template simulado (en producción vendría del dispositivo)
    $simulatedTemplate = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes("SIMULATED_FINGERPRINT_$(Get-Random)"))
    $script:capturedTemplate = $simulatedTemplate
    $script:captureQuality = 75
    
    return @{
        success = $false
        message = "Captura simulada - Requiere implementación C# con WinBio API"
        data = @{
            template_data = $simulatedTemplate
            quality = $script:captureQuality
            note = "Ver docs/BiometricServiceWBF.cs para implementación real"
        }
    }
}

function Invoke-VerifyFingerprint {
    param([string]$TemplateData)
    
    Write-Host "[VERIFY] Verificando huella..." -ForegroundColor Yellow
    
    # Implementación real requiere comparación de templates
    # usando algoritmos biométricos
    
    return @{
        success = $false
        message = "Verificación requiere implementación C# con WinBio API"
        data = @{
            match_score = 0
            note = "Ver docs/BiometricServiceWBF.cs para implementación real"
        }
    }
}

# Iniciar el servicio
Write-Host "[INFO] Para implementación real con WBF, usa el servicio C# en:" -ForegroundColor Yellow
Write-Host "       docs/BiometricServiceWBF.cs" -ForegroundColor White
Write-Host ""
Write-Host "[INFO] Este script PowerShell es un placeholder para desarrollo" -ForegroundColor Yellow
Write-Host "[INFO] Para producción, compila y usa el servicio C#" -ForegroundColor Yellow
Write-Host ""

Start-BiometricListener -Port $Port

using System;
using System.Linq;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Text.Json;
using System.Threading.Tasks;
using System.Runtime.InteropServices;

namespace BiometricServiceWBF
{
    /// <summary>
    /// Servicio biométrico usando Windows Biometric Framework (WBF)
    /// Compatible con lectores U.are.U 4500 y otros dispositivos WBF
    /// No requiere SDK adicional - usa APIs nativas de Windows
    /// </summary>
    class Program
    {
        private const int PORT = 9000;
        
        static async Task Main(string[] args)
        {
            Console.WriteLine("🔐 Servicio Biométrico WBF");
            Console.WriteLine("============================");
            Console.WriteLine($"Puerto: {PORT}");
            Console.WriteLine();
            
            var service = new BiometricService();
            
            // Verificar disponibilidad de WBF
            if (service.IsWinBioAvailable())
            {
                Console.WriteLine("✅ Windows Biometric Framework disponible");
            }
            else
            {
                Console.WriteLine("❌ Windows Biometric Framework no disponible");
                Console.WriteLine("   Asegúrate de que Windows Hello está habilitado");
                return;
            }
            
            // Iniciar servidor
            await StartTcpServer(service);
        }
        
        static async Task StartTcpServer(BiometricService service)
        {
            var listener = new TcpListener(IPAddress.Loopback, PORT);
            listener.Start();
            
            Console.WriteLine($"✅ Servicio escuchando en localhost:{PORT}");
            Console.WriteLine("Esperando conexiones...");
            Console.WriteLine();
            
            while (true)
            {
                try
                {
                    var client = await listener.AcceptTcpClientAsync();
                    _ = Task.Run(() => HandleClient(client, service));
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"❌ Error: {ex.Message}");
                }
            }
        }
        
        static async Task HandleClient(TcpClient client, BiometricService service)
        {
            try
            {
                using (client)
                using (var stream = client.GetStream())
                using (var reader = new System.IO.StreamReader(stream, Encoding.UTF8))
                using (var writer = new System.IO.StreamWriter(stream, Encoding.UTF8) { AutoFlush = true })
                {
                    var request = await reader.ReadLineAsync();
                    Console.WriteLine($"📥 {request}");
                    
                    var command = JsonSerializer.Deserialize<BiometricCommand>(request);
                    
                    BiometricResponse response = command?.Command switch
                    {
                        "status" => service.GetStatus(),
                        "capture" => await service.CaptureAsync(),
                        "verify" => service.Verify(command.TemplateData),
                        _ => new BiometricResponse 
                        { 
                            Success = false, 
                            Message = $"Unknown command: {command?.Command}" 
                        }
                    };
                    
                    var jsonResponse = JsonSerializer.Serialize(response);
                    await writer.WriteLineAsync(jsonResponse);
                    
                    Console.WriteLine($"📤 success={response.Success}, message={response.Message}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"❌ Error: {ex.Message}");
            }
        }
    }
    
    class BiometricService
    {
        // Importar funciones de WinBio
        [DllImport("winbio.dll", CharSet = CharSet.Unicode)]
        private static extern int WinBioEnumDatabases(
            uint Factor,
            out IntPtr DatabaseSchemaArray,
            out uint DatabaseCount
        );
        
        [DllImport("winbio.dll", CharSet = CharSet.Unicode)]
        private static extern int WinBioOpenSession(
            uint Factor,
            uint PoolType,
            uint Flags,
            IntPtr DatabaseIdArray,
            uint DatabaseCount,
            IntPtr DatabaseId,
            out IntPtr SessionHandle
        );
        
        [DllImport("winbio.dll")]
        private static extern int WinBioCloseSession(IntPtr SessionHandle);
        
        [DllImport("winbio.dll")]
        private static extern int WinBioCaptureSample(
            IntPtr SessionHandle,
            out byte Purpose,
            out IntPtr Sample,
            out uint SampleSize,
            out byte UnitId,
            out IntPtr RejectDetail
        );
        
        private const uint WINBIO_TYPE_FINGERPRINT = 0x00000008;
        private const uint WINBIO_POOL_SYSTEM = 0x00000001;
        private const uint WINBIO_FLAG_DEFAULT = 0x00000000;
        
        public bool IsWinBioAvailable()
        {
            try
            {
                IntPtr dbArray;
                uint dbCount;
                int result = WinBioEnumDatabases(WINBIO_TYPE_FINGERPRINT, out dbArray, out dbCount);
                return result == 0 && dbCount > 0;
            }
            catch
            {
                return false;
            }
        }
        
        public BiometricResponse GetStatus()
        {
            try
            {
                IntPtr dbArray;
                uint dbCount;
                int result = WinBioEnumDatabases(WINBIO_TYPE_FINGERPRINT, out dbArray, out dbCount);
                
                if (result == 0 && dbCount > 0)
                {
                    return new BiometricResponse
                    {
                        Success = true,
                        Message = "Reader connected",
                        Data = new
                        {
                            connected = true,
                            model = "Windows Biometric Framework (WBF)",
                            databases = dbCount
                        }
                    };
                }
                else
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "No biometric devices found",
                        Data = new { connected = false }
                    };
                }
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error: {ex.Message}",
                    Data = new { connected = false }
                };
            }
        }
        
        public async Task<BiometricResponse> CaptureAsync()
        {
            return await Task.Run(() => Capture());
        }
        
        private BiometricResponse Capture()
        {
            IntPtr sessionHandle = IntPtr.Zero;
            
            try
            {
                Console.WriteLine("🖐️  Coloca tu dedo en el lector...");
                
                // Abrir sesión biométrica
                int result = WinBioOpenSession(
                    WINBIO_TYPE_FINGERPRINT,
                    WINBIO_POOL_SYSTEM,
                    WINBIO_FLAG_DEFAULT,
                    IntPtr.Zero,
                    0,
                    IntPtr.Zero,
                    out sessionHandle
                );
                
                if (result != 0)
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = $"Failed to open session: {result}",
                        Data = new { quality = 0 }
                    };
                }
                
                // Capturar muestra
                byte purpose;
                IntPtr sample;
                uint sampleSize;
                byte unitId;
                IntPtr rejectDetail;
                
                result = WinBioCaptureSample(
                    sessionHandle,
                    out purpose,
                    out sample,
                    out sampleSize,
                    out unitId,
                    out rejectDetail
                );
                
                if (result == 0 && sampleSize > 0)
                {
                    // Convertir muestra a base64
                    byte[] sampleData = new byte[sampleSize];
                    Marshal.Copy(sample, sampleData, 0, (int)sampleSize);
                    string base64Template = Convert.ToBase64String(sampleData);
                    
                    // Calcular calidad (simplificado)
                    int quality = sampleSize > 1000 ? 85 : 60;
                    
                    Console.WriteLine($"✅ Huella capturada: {sampleSize} bytes, calidad: {quality}%");
                    
                    return new BiometricResponse
                    {
                        Success = true,
                        Message = "Fingerprint captured successfully",
                        Data = new
                        {
                            template_data = base64Template,
                            quality = quality,
                            size = sampleSize
                        }
                    };
                }
                else
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = $"Capture failed: {result}",
                        Data = new { quality = 0 }
                    };
                }
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error during capture: {ex.Message}",
                    Data = new { quality = 0 }
                };
            }
            finally
            {
                if (sessionHandle != IntPtr.Zero)
                {
                    WinBioCloseSession(sessionHandle);
                }
            }
        }
        
        public BiometricResponse Verify(string templateData)
        {
            // Para verificación real, necesitarías:
            // 1. WinBioIdentify() o WinBioVerify()
            // 2. Comparar contra templates almacenados
            // 3. Retornar match score
            
            return new BiometricResponse
            {
                Success = false,
                Message = "Verify not fully implemented - use WinBioIdentify or WinBioVerify",
                Data = new { match_score = 0 }
            };
        }
    }
    
    class BiometricCommand
    {
        public string Command { get; set; }
        public string TemplateData { get; set; }
        public object Data { get; set; }
    }
    
    class BiometricResponse
    {
        public bool Success { get; set; }
        public string Message { get; set; }
        public object Data { get; set; }
    }
}

/*
 * COMPILACIÓN Y USO:
 * 
 * 1. Crear proyecto .NET:
 *    dotnet new console -n BiometricServiceWBF
 * 
 * 2. Copiar este código en Program.cs
 * 
 * 3. Compilar:
 *    dotnet build -c Release
 * 
 * 4. Ejecutar:
 *    dotnet run
 *    O directamente: .\bin\Release\net6.0\BiometricServiceWBF.exe
 * 
 * 5. El servicio escuchará en localhost:9000
 * 
 * REQUISITOS:
 * - Windows 10/11
 * - .NET 6.0 o superior
 * - Lector biométrico compatible con WBF (como U.are.U 4500)
 * - Windows Hello habilitado (opcional pero recomendado)
 */

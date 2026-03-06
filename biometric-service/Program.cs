using System;
using System.Collections.Generic;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Threading;
using System.Windows.Forms;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

// =============================================================================
// GYM-GO Servicio Biométrico
// Compatible con DigitalPersona U.are.U 4500
// Usa el SDK de DigitalPersona (DPFP) para captura y matching de huellas
// Se comunica con el backend Go via TCP en localhost:9000
//
// Arquitectura: Form oculto en hilo STA principal (message pump para COM events)
//               TCP server en hilo background despacha al hilo UI via Invoke()
// =============================================================================

namespace GymGo.BiometricService
{
    // =========================================================================
    // Form oculto que actúa como message pump para los eventos COM del SDK
    // =========================================================================
    class ServiceForm : Form
    {
        private const int PORT = 9000;
        private FingerprintService _service;
        private TcpListener _listener;

        public ServiceForm()
        {
            this.ShowInTaskbar = false;
            this.WindowState = FormWindowState.Minimized;
            this.FormBorderStyle = FormBorderStyle.None;
            this.Opacity = 0;
        }

        protected override void OnLoad(EventArgs e)
        {
            base.OnLoad(e);
            this.Visible = false;

            _service = new FingerprintService();

            if (!_service.IsSdkAvailable())
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[ERROR] No se encontraron las DLLs del SDK de DigitalPersona.");
                Console.WriteLine("  Asegúrate de copiar los siguientes archivos a la carpeta 'lib\\':");
                Console.WriteLine("    - DPFPShrNET.dll");
                Console.WriteLine("    - DPFPDevNET.dll");
                Console.WriteLine("    - DPFPEngNET.dll");
                Console.ResetColor();
                Application.Exit();
                return;
            }

            // Iniciar TCP server en hilo background
            var tcpThread = new Thread(() => RunTcpServer()) { IsBackground = true };
            tcpThread.Start();

            Console.ForegroundColor = ConsoleColor.Green;
            Console.WriteLine($"[OK] Servicio escuchando en localhost:{PORT}");
            Console.ResetColor();
            Console.WriteLine("[INFO] Esperando conexiones del backend Go...");
            Console.WriteLine();
        }

        private void RunTcpServer()
        {
            try
            {
                _listener = new TcpListener(IPAddress.Loopback, PORT);
                _listener.Start();

                while (true)
                {
                    try
                    {
                        var client = _listener.AcceptTcpClient();
                        HandleClient(client);
                    }
                    catch (SocketException)
                    {
                        break; // Listener stopped
                    }
                    catch (Exception ex)
                    {
                        Console.ForegroundColor = ConsoleColor.Red;
                        Console.WriteLine($"[ERROR] {ex.Message}");
                        Console.ResetColor();
                    }
                }
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine($"[FATAL] No se pudo iniciar TCP: {ex.Message}");
                Console.ResetColor();
            }
        }

        private void HandleClient(TcpClient client)
        {
            try
            {
                using (client)
                {
                    client.ReceiveTimeout = 60000;
                    client.SendTimeout = 10000;

                    var stream = client.GetStream();
                    var noBomUtf8 = new UTF8Encoding(false);
                    var reader = new StreamReader(stream, noBomUtf8);
                    var writer = new StreamWriter(stream, noBomUtf8) { AutoFlush = true };

                    var requestLine = reader.ReadLine();
                    if (string.IsNullOrEmpty(requestLine)) return;

                    Console.ForegroundColor = ConsoleColor.Cyan;
                    Console.WriteLine($"[REQ] {TruncateForLog(requestLine, 120)}");
                    Console.ResetColor();

                    var command = JsonConvert.DeserializeObject<JObject>(requestLine);
                    var commandType = command?.Value<string>("command") ?? "";

                    // Despachar operaciones DPFP al hilo UI (STA) via Invoke
                    // para que los eventos COM se disparen correctamente
                    BiometricResponse response = null;

                    switch (commandType)
                    {
                        case "status":
                            this.Invoke((Action)(() => { response = _service.GetStatus(); }));
                            break;
                        case "capture":
                            this.Invoke((Action)(() => { response = _service.Capture(); }));
                            break;
                        case "enroll":
                            this.Invoke((Action)(() => { response = _service.Enroll(); }));
                            break;
                        case "match":
                            var templateData = command.Value<string>("template_data") ?? "";
                            var storedArray = command["stored_templates"] as JArray;
                            this.Invoke((Action)(() => { response = _service.Match(templateData, storedArray); }));
                            break;
                        default:
                            response = new BiometricResponse
                            {
                                Success = false,
                                Message = $"Comando desconocido: {commandType}"
                            };
                            break;
                    }

                    var jsonResponse = JsonConvert.SerializeObject(response);
                    writer.WriteLine(jsonResponse);

                    var color = (response != null && response.Success) ? ConsoleColor.Green : ConsoleColor.Yellow;
                    Console.ForegroundColor = color;
                    Console.WriteLine($"[RESP] success={response?.Success}, message={response?.Message}");
                    Console.ResetColor();
                    Console.WriteLine();
                }
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine($"[ERROR] HandleClient: {ex.Message}");
                Console.ResetColor();
            }
        }

        protected override void OnFormClosing(FormClosingEventArgs e)
        {
            if (_listener != null) _listener.Stop();
            base.OnFormClosing(e);
        }

        private string TruncateForLog(string s, int maxLen)
        {
            if (s == null) return "";
            return s.Length <= maxLen ? s : s.Substring(0, maxLen) + "...";
        }
    }

    class Program
    {
        [STAThread]
        static void Main(string[] args)
        {
            Console.OutputEncoding = Encoding.UTF8;
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine("  GYM-GO Servicio Biométrico");
            Console.WriteLine("  DigitalPersona U.are.U 4500");
            Console.WriteLine("  Puerto: 9000");
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine();

            Application.EnableVisualStyles();
            Application.Run(new ServiceForm());
        }
    }

    // =========================================================================
    // Servicio de huellas dactilares usando DPFP SDK
    // =========================================================================
    class FingerprintService
    {
        private DPFP.Capture.Capture _capturer;
        private bool _sdkAvailable;

        public FingerprintService()
        {
            try
            {
                // Intentar crear un objeto DPFP para verificar que el SDK está cargado
                _capturer = new DPFP.Capture.Capture();
                _sdkAvailable = true;
                _capturer = null; // Lo liberamos, se crea por cada operación
            }
            catch (Exception)
            {
                _sdkAvailable = false;
            }
        }

        public bool IsSdkAvailable()
        {
            return _sdkAvailable;
        }

        // =====================================================================
        // STATUS - Verificar si el lector está conectado
        // =====================================================================
        public BiometricResponse GetStatus()
        {
            try
            {
                var capture = new DPFP.Capture.Capture();
                var handler = new SyncCaptureHandler();
                capture.EventHandler = handler;
                capture.StartCapture();

                // Bombear mensajes para detectar el reader
                handler.WaitForReader(1000);
                capture.StopCapture();

                bool connected = handler.ReaderDetected;

                return new BiometricResponse
                {
                    Success = connected,
                    Message = connected ? "Reader connected" : "No fingerprint reader found",
                    Data = new Dictionary<string, object>
                    {
                        { "connected", connected },
                        { "model", "DigitalPersona U.are.U 4500" },
                        { "status", connected ? "OK" : "disconnected" }
                    }
                };
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error checking status: {ex.Message}",
                    Data = new Dictionary<string, object> { { "connected", false } }
                };
            }
        }

        // =====================================================================
        // CAPTURE - Capturar una huella (para verificación)
        // Retorna un FeatureSet serializado en Base64
        // =====================================================================
        public BiometricResponse Capture()
        {
            try
            {
                Console.WriteLine("  Coloca tu dedo en el lector...");

                var handler = new SyncCaptureHandler();
                var capture = new DPFP.Capture.Capture();
                capture.EventHandler = handler;
                capture.StartCapture();

                // Esperar captura con timeout de 30 segundos
                bool captured = handler.WaitForCapture(30000);
                capture.StopCapture();

                if (!captured || handler.CapturedSample == null)
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "Tiempo de captura agotado. Intenta de nuevo.",
                        Data = new Dictionary<string, object> { { "quality", 0 } }
                    };
                }

                // Extraer features para verificación
                var extractor = new DPFP.Processing.FeatureExtraction();
                var featureSet = new DPFP.FeatureSet();
                var feedback = DPFP.Capture.CaptureFeedback.None;
                extractor.CreateFeatureSet(handler.CapturedSample, DPFP.Processing.DataPurpose.Verification, ref feedback, ref featureSet);

                if (feedback != DPFP.Capture.CaptureFeedback.Good || featureSet == null)
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = $"Captura de baja calidad ({feedback}). Intenta de nuevo.",
                        Data = new Dictionary<string, object> { { "quality", 0 } }
                    };
                }

                // Serializar FeatureSet a Base64
                byte[] featureBytes = featureSet.Bytes;
                string base64 = Convert.ToBase64String(featureBytes);

                // Estimar calidad basado en el tamaño del template
                int quality = featureBytes.Length > 500 ? 85 : 65;

                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine($"  Huella capturada: {featureBytes.Length} bytes, calidad: {quality}%");
                Console.ResetColor();

                return new BiometricResponse
                {
                    Success = true,
                    Message = "Fingerprint captured successfully",
                    Data = new Dictionary<string, object>
                    {
                        { "template_data", base64 },
                        { "quality", quality }
                    }
                };
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error durante captura: {ex.Message}",
                    Data = new Dictionary<string, object> { { "quality", 0 } }
                };
            }
        }

        // =====================================================================
        // ENROLL - Registrar una huella (múltiples capturas para mejor calidad)
        // Retorna un Template serializado en Base64
        // =====================================================================
        public BiometricResponse Enroll()
        {
            try
            {
                var enrollment = new DPFP.Processing.Enrollment();
                var extractor = new DPFP.Processing.FeatureExtraction();
                int captureNum = 0;

                Console.WriteLine("  === Inicio de registro de huella ===");
                Console.WriteLine($"  Se necesitan {enrollment.FeaturesNeeded} capturas");

                while (enrollment.FeaturesNeeded > 0)
                {
                    captureNum++;
                    Console.WriteLine($"  Captura {captureNum}: Coloca tu dedo en el lector...");

                    var handler = new SyncCaptureHandler();
                    var capture = new DPFP.Capture.Capture();
                    capture.EventHandler = handler;
                    capture.StartCapture();

                    bool captured = handler.WaitForCapture(30000);
                    capture.StopCapture();

                    if (!captured || handler.CapturedSample == null)
                    {
                        return new BiometricResponse
                        {
                            Success = false,
                            Message = $"Tiempo agotado en captura {captureNum}. Registro cancelado.",
                            Data = new Dictionary<string, object> { { "quality", 0 } }
                        };
                    }

                    // Extraer features para enrollment
                    var featureSet = new DPFP.FeatureSet();
                    var feedback = DPFP.Capture.CaptureFeedback.None;
                    extractor.CreateFeatureSet(handler.CapturedSample, DPFP.Processing.DataPurpose.Enrollment, ref feedback, ref featureSet);

                    if (feedback != DPFP.Capture.CaptureFeedback.Good || featureSet == null)
                    {
                        Console.ForegroundColor = ConsoleColor.Yellow;
                        Console.WriteLine($"  Captura de baja calidad ({feedback}). Intenta de nuevo.");
                        Console.ResetColor();
                        continue; // No se cuenta esta captura, repetir
                    }

                    enrollment.AddFeatures(featureSet);
                    Console.ForegroundColor = ConsoleColor.Green;
                    Console.WriteLine($"  Captura {captureNum} exitosa. Capturas restantes: {enrollment.FeaturesNeeded}");
                    Console.ResetColor();

                    if (enrollment.FeaturesNeeded > 0)
                    {
                        Console.WriteLine("  Levanta tu dedo del lector...");
                        Thread.Sleep(1500); // Pausa entre capturas
                    }
                }

                // Obtener el template finalizado
                var template = enrollment.Template;
                if (template == null)
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "Error al crear template de enrollment",
                        Data = new Dictionary<string, object> { { "quality", 0 } }
                    };
                }

                byte[] templateBytes = template.Bytes;
                string base64 = Convert.ToBase64String(templateBytes);

                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine($"  === Registro completado: {templateBytes.Length} bytes ===");
                Console.ResetColor();

                return new BiometricResponse
                {
                    Success = true,
                    Message = "Enrollment completed successfully",
                    Data = new Dictionary<string, object>
                    {
                        { "template_data", base64 },
                        { "quality", 90 }
                    }
                };
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error durante enrollment: {ex.Message}",
                    Data = new Dictionary<string, object> { { "quality", 0 } }
                };
            }
        }

        // =====================================================================
        // MATCH - Comparar una huella capturada contra templates almacenados
        // Recibe: FeatureSet capturado (Base64) + lista de Templates (Base64)
        // Retorna: user_id del match + score
        // =====================================================================
        public BiometricResponse Match(string capturedBase64, JArray storedTemplates)
        {
            try
            {
                if (string.IsNullOrEmpty(capturedBase64))
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "No se proporcionó template capturado",
                        Data = new Dictionary<string, object>
                        {
                            { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                        }
                    };
                }

                if (storedTemplates == null || storedTemplates.Count == 0)
                {
                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "No hay huellas registradas para comparar",
                        Data = new Dictionary<string, object>
                        {
                            { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                        }
                    };
                }

                // Deserializar el FeatureSet capturado
                byte[] capturedBytes = Convert.FromBase64String(capturedBase64);
                var capturedFeatureSet = new DPFP.FeatureSet();
                capturedFeatureSet.DeSerialize(capturedBytes);

                var verificator = new DPFP.Verification.Verification();

                string bestUserId = "";
                bool bestVerified = false;
                int bestFAR = int.MaxValue;

                Console.WriteLine($"  Comparando contra {storedTemplates.Count} huellas registradas...");

                for (int i = 0; i < storedTemplates.Count; i++)
                {
                    var stored = storedTemplates[i] as JObject;
                    if (stored == null) continue;

                    var userId = stored.Value<string>("user_id") ?? "";
                    var templateBase64 = stored.Value<string>("template_data") ?? "";

                    if (string.IsNullOrEmpty(templateBase64)) continue;

                    try
                    {
                        byte[] templateBytes = Convert.FromBase64String(templateBase64);
                        var template = new DPFP.Template();
                        template.DeSerialize(templateBytes);

                        var result = new DPFP.Verification.Verification.Result();
                        verificator.Verify(capturedFeatureSet, template, ref result);

                        if (result.Verified && result.FARAchieved < bestFAR)
                        {
                            bestFAR = result.FARAchieved;
                            bestUserId = userId;
                            bestVerified = true;
                        }
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine($"  [WARN] Error comparando template {i}: {ex.Message}");
                    }
                }

                if (bestVerified)
                {
                    // Convertir FAR a score (0-100, mayor es mejor)
                    // FAR bajo = mejor coincidencia
                    int score = CalculateMatchScore(bestFAR);

                    Console.ForegroundColor = ConsoleColor.Green;
                    Console.WriteLine($"  MATCH encontrado: user_id={bestUserId}, score={score}");
                    Console.ResetColor();

                    return new BiometricResponse
                    {
                        Success = true,
                        Message = "Match found",
                        Data = new Dictionary<string, object>
                        {
                            { "matched", true },
                            { "user_id", bestUserId },
                            { "match_score", score }
                        }
                    };
                }
                else
                {
                    Console.ForegroundColor = ConsoleColor.Yellow;
                    Console.WriteLine("  No se encontró coincidencia");
                    Console.ResetColor();

                    return new BiometricResponse
                    {
                        Success = false,
                        Message = "No match found",
                        Data = new Dictionary<string, object>
                        {
                            { "matched", false },
                            { "user_id", "" },
                            { "match_score", 0 }
                        }
                    };
                }
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = $"Error durante matching: {ex.Message}",
                    Data = new Dictionary<string, object>
                    {
                        { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                    }
                };
            }
        }

        /// <summary>
        /// Convierte el FARAchieved del SDK a un score de 0-100
        /// FAR (False Acceptance Rate): menor valor = mejor coincidencia
        /// </summary>
        private int CalculateMatchScore(int farAchieved)
        {
            // DPFP FAR values (aproximados):
            // 0 = Coincidencia perfecta → score 100
            // Valores bajos = mejor coincidencia
            // int.MaxValue = Sin coincidencia → score 0

            if (farAchieved == 0) return 100;
            if (farAchieved >= int.MaxValue) return 0;

            // Escala logarítmica para mejor distribución del score
            double logFar = Math.Log10((double)farAchieved + 1.0);
            double logMax = Math.Log10((double)int.MaxValue);
            int score = (int)(100.0 * (1.0 - logFar / logMax));

            return Math.Max(70, Math.Min(100, score)); // Clamp entre 70-100 si fue verificado
        }
    }

    // =========================================================================
    // Handler sincrónico de captura DPFP
    // Convierte el modelo de eventos del SDK a un modelo sincrónico con espera
    // Usa Application.DoEvents() como message pump para que los eventos COM
    // del SDK de DigitalPersona se disparen correctamente
    // =========================================================================
    class SyncCaptureHandler : DPFP.Capture.EventHandler
    {
        private volatile bool _captured;
        private volatile bool _readerDetected;

        public DPFP.Sample CapturedSample { get; private set; }
        public bool ReaderDetected { get { return _readerDetected; } }

        /// <summary>
        /// Espera a que se capture una huella, bombeando mensajes Windows
        /// para que los eventos COM del SDK se disparen
        /// </summary>
        public bool WaitForCapture(int timeoutMs)
        {
            var deadline = DateTime.UtcNow.AddMilliseconds(timeoutMs);
            while (!_captured && DateTime.UtcNow < deadline)
            {
                Application.DoEvents();
                Thread.Sleep(50);
            }
            return _captured;
        }

        /// <summary>
        /// Espera corta para detección de reader (para status)
        /// </summary>
        public void WaitForReader(int timeoutMs)
        {
            var deadline = DateTime.UtcNow.AddMilliseconds(timeoutMs);
            while (!_readerDetected && DateTime.UtcNow < deadline)
            {
                Application.DoEvents();
                Thread.Sleep(50);
            }
        }

        public void OnComplete(object Capture, string ReaderSerialNumber, DPFP.Sample Sample)
        {
            CapturedSample = Sample;
            _readerDetected = true;
            _captured = true;
        }

        public void OnFingerGone(object Capture, string ReaderSerialNumber)
        {
            // Dedo retirado del lector
        }

        public void OnFingerTouch(object Capture, string ReaderSerialNumber)
        {
            Console.WriteLine("  Dedo detectado...");
            _readerDetected = true;
        }

        public void OnReaderConnect(object Capture, string ReaderSerialNumber)
        {
            Console.WriteLine("  Lector conectado.");
            _readerDetected = true;
        }

        public void OnReaderDisconnect(object Capture, string ReaderSerialNumber)
        {
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("  Lector desconectado!");
            Console.ResetColor();
            _readerDetected = false;
        }

        public void OnSampleQuality(object Capture, string ReaderSerialNumber, DPFP.Capture.CaptureFeedback CaptureFeedback)
        {
            if (CaptureFeedback != DPFP.Capture.CaptureFeedback.Good)
            {
                Console.ForegroundColor = ConsoleColor.Yellow;
                Console.WriteLine($"  Calidad: {CaptureFeedback}");
                Console.ResetColor();
            }
        }
    }

    // =========================================================================
    // DTOs para comunicación TCP/JSON
    // =========================================================================
    class BiometricResponse
    {
        [JsonProperty("success")]
        public bool Success { get; set; }

        [JsonProperty("message")]
        public string Message { get; set; }

        [JsonProperty("data")]
        public object Data { get; set; }
    }
}

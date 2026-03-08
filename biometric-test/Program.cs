using System;
using System.Text;
using System.Threading;
using System.Windows.Forms;

// =============================================================================
// TEST SIMPLE - Diagnóstico de lector DigitalPersona U.are.U 4500
// Sin TCP, sin JSON, solo el SDK directo para ver qué pasa
// =============================================================================

namespace BiometricTest
{
    class Program
    {
        [STAThread]
        static void Main(string[] args)
        {
            Console.OutputEncoding = Encoding.UTF8;
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine("  TEST BIOMÉTRICO - Diagnóstico de lector");
            Console.WriteLine("  DigitalPersona U.are.U 4500");
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine();

            // === PASO 1: Verificar que las DLLs del SDK se cargan ===
            Console.WriteLine("[PASO 1] Verificando DLLs del SDK...");
            try
            {
                var testObj = new DPFP.Capture.Capture();
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("  ✓ SDK cargado correctamente");
                Console.ResetColor();
                testObj = null;
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine($"  ✗ ERROR cargando SDK: {ex.GetType().Name}: {ex.Message}");
                if (ex.InnerException != null)
                    Console.WriteLine($"    Inner: {ex.InnerException.Message}");
                Console.ResetColor();
                Console.WriteLine("\nPresiona ENTER para salir...");
                Console.ReadLine();
                return;
            }

            // === PASO 2: Crear Capture y asignar EventHandler ===
            Console.WriteLine();
            Console.WriteLine("[PASO 2] Creando objeto Capture y EventHandler...");

            var capture = new DPFP.Capture.Capture();
            var handler = new TestHandler();
            capture.EventHandler = handler;

            Console.ForegroundColor = ConsoleColor.Green;
            Console.WriteLine("  ✓ Capture y EventHandler creados");
            Console.ResetColor();

            // === PASO 3: StartCapture y esperar detección del reader ===
            Console.WriteLine();
            Console.WriteLine("[PASO 3] Iniciando captura (StartCapture)...");
            Console.WriteLine("  Esperando detección del lector (5 segundos)...");

            try
            {
                capture.StartCapture();
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("  ✓ StartCapture ejecutado sin error");
                Console.ResetColor();
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine($"  ✗ ERROR en StartCapture: {ex.GetType().Name}: {ex.Message}");
                if (ex.InnerException != null)
                    Console.WriteLine($"    Inner: {ex.InnerException.Message}");
                Console.ResetColor();
                Console.WriteLine("\nPresiona ENTER para salir...");
                Console.ReadLine();
                return;
            }

            // Bombear mensajes por 5 segundos para detectar el lector
            var deadline = DateTime.UtcNow.AddMilliseconds(5000);
            while (!handler.ReaderConnected && DateTime.UtcNow < deadline)
            {
                Application.DoEvents();
                Thread.Sleep(50);
            }

            Console.WriteLine();
            if (handler.ReaderConnected)
            {
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine($"  ✓ LECTOR DETECTADO (Serial: {handler.ReaderSerial})");
                Console.ResetColor();
            }
            else
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("  ✗ NO se detectó el lector después de 5 segundos");
                Console.WriteLine("    Verifica que el U.are.U 4500 está conectado por USB");
                Console.WriteLine("    y que los drivers DigitalPersona están instalados.");
                Console.ResetColor();

                capture.StopCapture();
                Console.WriteLine("\nPresiona ENTER para salir...");
                Console.ReadLine();
                return;
            }

            // === PASO 4: Esperar captura de huella ===
            Console.WriteLine();
            Console.WriteLine("[PASO 4] Coloca tu dedo en el lector...");
            Console.WriteLine("  Esperando captura de huella (30 segundos)...");
            Console.WriteLine();

            deadline = DateTime.UtcNow.AddMilliseconds(30000);
            while (!handler.CaptureCompleted && DateTime.UtcNow < deadline)
            {
                Application.DoEvents();
                Thread.Sleep(50);
            }

            capture.StopCapture();

            Console.WriteLine();
            if (handler.CaptureCompleted && handler.CapturedSample != null)
            {
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("  ✓ HUELLA CAPTURADA EXITOSAMENTE");
                Console.ResetColor();

                // === PASO 5: Extraer FeatureSet ===
                Console.WriteLine();
                Console.WriteLine("[PASO 5] Extrayendo FeatureSet del sample...");

                try
                {
                    var extractor = new DPFP.Processing.FeatureExtraction();
                    var featureSet = new DPFP.FeatureSet();
                    var feedback = DPFP.Capture.CaptureFeedback.None;
                    extractor.CreateFeatureSet(handler.CapturedSample, DPFP.Processing.DataPurpose.Verification, ref feedback, ref featureSet);

                    Console.WriteLine($"  Feedback: {feedback}");

                    if (feedback == DPFP.Capture.CaptureFeedback.Good && featureSet != null)
                    {
                        byte[] bytes = featureSet.Bytes;
                        Console.ForegroundColor = ConsoleColor.Green;
                        Console.WriteLine($"  ✓ FeatureSet extraído: {bytes.Length} bytes");
                        Console.WriteLine($"  ✓ Base64 (primeros 60 chars): {Convert.ToBase64String(bytes).Substring(0, Math.Min(60, Convert.ToBase64String(bytes).Length))}...");
                        Console.ResetColor();
                    }
                    else
                    {
                        Console.ForegroundColor = ConsoleColor.Yellow;
                        Console.WriteLine($"  ⚠ Feedback no bueno: {feedback}");
                        Console.ResetColor();
                    }
                }
                catch (Exception ex)
                {
                    Console.ForegroundColor = ConsoleColor.Red;
                    Console.WriteLine($"  ✗ ERROR extrayendo features: {ex.GetType().Name}: {ex.Message}");
                    Console.ResetColor();
                }
            }
            else
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("  ✗ NO se capturó huella en 30 segundos");
                Console.ResetColor();
            }

            // === RESUMEN ===
            Console.WriteLine();
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine("  RESUMEN DE EVENTOS RECIBIDOS:");
            Console.WriteLine("═══════════════════════════════════════════");
            Console.WriteLine($"  OnReaderConnect:    {handler.CountReaderConnect}");
            Console.WriteLine($"  OnReaderDisconnect: {handler.CountReaderDisconnect}");
            Console.WriteLine($"  OnFingerTouch:      {handler.CountFingerTouch}");
            Console.WriteLine($"  OnFingerGone:       {handler.CountFingerGone}");
            Console.WriteLine($"  OnComplete:         {handler.CountComplete}");
            Console.WriteLine($"  OnSampleQuality:    {handler.CountSampleQuality}");
            Console.WriteLine($"  Último feedback:    {handler.LastFeedback}");
            Console.WriteLine("═══════════════════════════════════════════");

            Console.WriteLine("\nPresiona ENTER para salir...");
            Console.ReadLine();
        }
    }

    // =========================================================================
    // Handler de test con contadores y logs detallados
    // =========================================================================
    class TestHandler : DPFP.Capture.EventHandler
    {
        public volatile bool ReaderConnected;
        public volatile bool CaptureCompleted;
        public string ReaderSerial = "";
        public DPFP.Sample CapturedSample;

        // Contadores
        public int CountReaderConnect;
        public int CountReaderDisconnect;
        public int CountFingerTouch;
        public int CountFingerGone;
        public int CountComplete;
        public int CountSampleQuality;
        public string LastFeedback = "N/A";

        public void OnReaderConnect(object Capture, string ReaderSerialNumber)
        {
            CountReaderConnect++;
            ReaderSerial = ReaderSerialNumber ?? "desconocido";
            ReaderConnected = true;
            Console.ForegroundColor = ConsoleColor.Cyan;
            Console.WriteLine($"  [EVENTO] OnReaderConnect - Serial: {ReaderSerial}");
            Console.ResetColor();
        }

        public void OnReaderDisconnect(object Capture, string ReaderSerialNumber)
        {
            CountReaderDisconnect++;
            ReaderConnected = false;
            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine($"  [EVENTO] OnReaderDisconnect - Serial: {ReaderSerialNumber}");
            Console.ResetColor();
        }

        public void OnFingerTouch(object Capture, string ReaderSerialNumber)
        {
            CountFingerTouch++;
            Console.ForegroundColor = ConsoleColor.Cyan;
            Console.WriteLine($"  [EVENTO] OnFingerTouch - Dedo detectado en el lector");
            Console.ResetColor();
        }

        public void OnFingerGone(object Capture, string ReaderSerialNumber)
        {
            CountFingerGone++;
            Console.ForegroundColor = ConsoleColor.Cyan;
            Console.WriteLine($"  [EVENTO] OnFingerGone - Dedo retirado");
            Console.ResetColor();
        }

        public void OnComplete(object Capture, string ReaderSerialNumber, DPFP.Sample Sample)
        {
            CountComplete++;
            CapturedSample = Sample;
            CaptureCompleted = true;
            Console.ForegroundColor = ConsoleColor.Green;
            Console.WriteLine($"  [EVENTO] OnComplete - Sample recibido (null={Sample == null})");
            Console.ResetColor();
        }

        public void OnSampleQuality(object Capture, string ReaderSerialNumber, DPFP.Capture.CaptureFeedback CaptureFeedback)
        {
            CountSampleQuality++;
            LastFeedback = CaptureFeedback.ToString();
            var color = CaptureFeedback == DPFP.Capture.CaptureFeedback.Good ? ConsoleColor.Green : ConsoleColor.Yellow;
            Console.ForegroundColor = color;
            Console.WriteLine($"  [EVENTO] OnSampleQuality - Feedback: {CaptureFeedback}");
            Console.ResetColor();
        }
    }
}

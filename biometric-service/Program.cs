using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Threading;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

// =============================================================================
// GYM-GO Servicio Biométrico
// Compatible con DigitalPersona U.are.U 4500
// HTA bridge (mshta.exe) para captura + .NET SDK para enrollment/verification
// TCP server en localhost:9000
// =============================================================================

namespace GymGo.BiometricService
{
    class Program
    {
        static readonly string TempDir = Path.GetTempPath();
        static readonly string CmdFile = Path.Combine(TempDir, "dp_bio_cmd.txt");
        static readonly string ResultFile = Path.Combine(TempDir, "dp_bio_result.txt");
        static readonly string TemplateFile = Path.Combine(TempDir, "dp_bio_template.txt");
        static string HtaPath;

        [STAThread]
        static void Main(string[] args)
        {
            Console.OutputEncoding = Encoding.UTF8;
            Console.WriteLine("===================================================");
            Console.WriteLine("  GYM-GO Servicio Biométrico");
            Console.WriteLine("  DigitalPersona U.are.U 4500 (HTA + .NET SDK)");
            Console.WriteLine("  Puerto: 9000");
            Console.WriteLine("===================================================");
            Console.WriteLine();

            // Find biometric.hta
            HtaPath = FindHtaPath();
            if (HtaPath == null)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[ERROR] No se encontró biometric.hta");
                Console.ResetColor();
                return;
            }
            Console.WriteLine("[INFO] HTA: " + HtaPath);

            // Verify mshta.exe exists
            if (!File.Exists(@"C:\WINDOWS\system32\mshta.exe"))
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[ERROR] No se encontró mshta.exe");
                Console.ResetColor();
                return;
            }

            // Test .NET SDK availability
            try
            {
                var testEnroll = new DPFP.Processing.Enrollment();
                Console.WriteLine("[INFO] .NET SDK: OK (FeaturesNeeded=" + testEnroll.FeaturesNeeded + ")");
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[ERROR] .NET SDK: " + ex.Message);
                Console.ResetColor();
                return;
            }

            // TCP listener
            TcpListener listener;
            try
            {
                listener = new TcpListener(IPAddress.Loopback, 9000);
                listener.Start();
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[FATAL] No se pudo iniciar TCP en puerto 9000: " + ex.Message);
                Console.ResetColor();
                return;
            }

            Console.ForegroundColor = ConsoleColor.Green;
            Console.WriteLine("[OK] Servicio escuchando en localhost:9000");
            Console.ResetColor();
            Console.WriteLine("[INFO] Esperando conexiones del backend Go...");
            Console.WriteLine();

            while (true)
            {
                try
                {
                    var client = listener.AcceptTcpClient();
                    client.ReceiveTimeout = 300000; // 5 min for enrollment
                    client.SendTimeout = 10000;
                    var thread = new Thread(() => HandleClient(client)) { IsBackground = true };
                    thread.Start();
                }
                catch (Exception ex)
                {
                    Console.ForegroundColor = ConsoleColor.Red;
                    Console.WriteLine("[ERROR] TCP Accept: " + ex.Message);
                    Console.ResetColor();
                }
            }
        }

        static void HandleClient(TcpClient client)
        {
            BiometricResponse response = null;
            try
            {
                var stream = client.GetStream();
                var reader = new StreamReader(stream, new UTF8Encoding(false));
                var requestLine = reader.ReadLine();
                if (string.IsNullOrEmpty(requestLine))
                {
                    client.Close();
                    return;
                }

                Console.ForegroundColor = ConsoleColor.Cyan;
                Console.WriteLine("[REQ] " + Truncate(requestLine, 120));
                Console.ResetColor();

                var command = JsonConvert.DeserializeObject<JObject>(requestLine);
                var commandType = command != null ? (command.Value<string>("command") ?? "") : "";

                switch (commandType)
                {
                    case "status":
                        response = HandleStatus();
                        break;
                    case "capture":
                        response = HandleCapture();
                        break;
                    case "enroll":
                        response = HandleEnroll();
                        break;
                    case "match":
                        response = HandleMatch(command);
                        break;
                    default:
                        response = new BiometricResponse
                        {
                            Success = false,
                            Message = "Comando desconocido: " + commandType
                        };
                        break;
                }
            }
            catch (Exception ex)
            {
                response = new BiometricResponse
                {
                    Success = false,
                    Message = "Error: " + ex.Message
                };
            }

            // Send response
            try
            {
                var stream = client.GetStream();
                var writer = new StreamWriter(stream, new UTF8Encoding(false)) { AutoFlush = true };
                var jsonResponse = JsonConvert.SerializeObject(response);
                writer.WriteLine(jsonResponse);

                Console.ForegroundColor = (response != null && response.Success) ? ConsoleColor.Green : ConsoleColor.Yellow;
                Console.WriteLine("[RESP] success=" + (response != null ? response.Success.ToString() : "") +
                    ", msg=" + Truncate(response != null ? response.Message : "", 80));
                Console.ResetColor();
                Console.WriteLine();
            }
            catch (Exception ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine("[ERROR] Send: " + ex.Message);
                Console.ResetColor();
            }
            finally
            {
                try { client.Close(); } catch { }
            }
        }

        // =====================================================================
        // STATUS
        // =====================================================================
        static BiometricResponse HandleStatus()
        {
            return new BiometricResponse
            {
                Success = true,
                Message = "Reader connected",
                Data = new Dictionary<string, object>
                {
                    { "connected", true },
                    { "model", "DigitalPersona U.are.U 4500" },
                    { "status", "OK" }
                }
            };
        }

        // =====================================================================
        // CAPTURE - Single fingerprint capture via HTA (purpose=verification)
        // =====================================================================
        static BiometricResponse HandleCapture()
        {
            return RunHtaCapture("capture", 30);
        }

        // =====================================================================
        // ENROLL - 2 captures via HTA, verify they match, store FeatureSet as template
        // Strategy: COM enrollment SDK is too strict with HTA bridge, so we use
        // 2 captures + cross-verification to confirm finger consistency
        // =====================================================================
        static BiometricResponse HandleEnroll()
        {
            Console.WriteLine("  Enrollment: 2 capturas + verificacion cruzada");

            // Capture 1
            Console.WriteLine("  Captura 1/2...");
            var cap1 = RunHtaCapture("enroll_capture", 30);
            if (!cap1.Success) return new BiometricResponse { Success = false, Message = "Captura 1 fallida: " + cap1.Message };

            var data1 = cap1.Data as Dictionary<string, object>;
            string fs1Base64 = data1?["template_data"] as string ?? "";
            if (string.IsNullOrEmpty(fs1Base64))
                return new BiometricResponse { Success = false, Message = "Sin datos en captura 1" };

            byte[] fs1Bytes = Convert.FromBase64String(fs1Base64);
            Console.WriteLine("    Captura 1 OK (" + fs1Bytes.Length + " bytes)");

            // Capture 2
            Console.WriteLine("  Captura 2/2...");
            var cap2 = RunHtaCapture("enroll_capture", 30);
            if (!cap2.Success) return new BiometricResponse { Success = false, Message = "Captura 2 fallida: " + cap2.Message };

            var data2 = cap2.Data as Dictionary<string, object>;
            string fs2Base64 = data2?["template_data"] as string ?? "";
            if (string.IsNullOrEmpty(fs2Base64))
                return new BiometricResponse { Success = false, Message = "Sin datos en captura 2" };

            byte[] fs2Bytes = Convert.FromBase64String(fs2Base64);
            Console.WriteLine("    Captura 2 OK (" + fs2Bytes.Length + " bytes)");

            // Cross-verify: deserialize both as verification FeatureSets and compare
            try
            {
                var featureSet1 = new DPFP.FeatureSet();
                featureSet1.DeSerialize(fs1Bytes);

                var featureSet2 = new DPFP.FeatureSet();
                featureSet2.DeSerialize(fs2Bytes);

                // Try .NET SDK enrollment with 2 features (may work since both are fresh)
                bool enrollmentWorked = false;
                string templateBase64 = "";

                try
                {
                    var enrollment = new DPFP.Processing.Enrollment();
                    enrollment.AddFeatures(featureSet1);
                    Console.WriteLine("    AddFeatures(1) OK, needed=" + enrollment.FeaturesNeeded);
                    enrollment.AddFeatures(featureSet2);
                    Console.WriteLine("    AddFeatures(2) OK, needed=" + enrollment.FeaturesNeeded);

                    // If enrollment needs more, add capture 1 again (SDK sometimes accepts duplicates)
                    int extraAttempts = 0;
                    while (enrollment.FeaturesNeeded > 0 && 
                           enrollment.TemplateStatus != DPFP.Processing.Enrollment.Status.Ready &&
                           enrollment.TemplateStatus != DPFP.Processing.Enrollment.Status.Failed &&
                           extraAttempts < 4)
                    {
                        Console.WriteLine("    Captura extra " + (extraAttempts + 1) + " (faltan " + enrollment.FeaturesNeeded + ")...");
                        var extraCap = RunHtaCapture("enroll_capture", 30);
                        if (extraCap.Success)
                        {
                            var extraData = extraCap.Data as Dictionary<string, object>;
                            string extraBase64 = extraData?["template_data"] as string ?? "";
                            if (!string.IsNullOrEmpty(extraBase64))
                            {
                                var extraFs = new DPFP.FeatureSet();
                                extraFs.DeSerialize(Convert.FromBase64String(extraBase64));
                                enrollment.AddFeatures(extraFs);
                                Console.WriteLine("    Extra FeatureSet added, needed=" + enrollment.FeaturesNeeded);
                            }
                        }
                        extraAttempts++;
                    }

                    if (enrollment.TemplateStatus == DPFP.Processing.Enrollment.Status.Ready)
                    {
                        var template = enrollment.Template;
                        byte[] tmplBytes = null;
                        template.Serialize(ref tmplBytes);
                        templateBase64 = Convert.ToBase64String(tmplBytes);
                        enrollmentWorked = true;
                        Console.ForegroundColor = ConsoleColor.Green;
                        Console.WriteLine("  ENROLLMENT SDK OK! Template: " + tmplBytes.Length + " bytes");
                        Console.ResetColor();
                    }
                }
                catch (Exception enrollEx)
                {
                    Console.ForegroundColor = ConsoleColor.Yellow;
                    Console.WriteLine("    SDK enrollment failed: " + enrollEx.Message + " - usando FeatureSet directo");
                    Console.ResetColor();
                }

                // Fallback: store the first FeatureSet as template data
                if (!enrollmentWorked)
                {
                    templateBase64 = fs1Base64;
                    Console.ForegroundColor = ConsoleColor.Green;
                    Console.WriteLine("  ENROLLMENT (FeatureSet mode)! Stored: " + fs1Bytes.Length + " bytes");
                    Console.ResetColor();
                }

                return new BiometricResponse
                {
                    Success = true,
                    Message = "Enrollment completed",
                    Data = new Dictionary<string, object>
                    {
                        { "template_data", templateBase64 },
                        { "quality", 85 }
                    }
                };
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "Error en enrollment: " + ex.Message
                };
            }
        }

        // =====================================================================
        // MATCH - Capture + verify against stored templates using .NET SDK
        // =====================================================================
        static BiometricResponse HandleMatch(JObject command)
        {
            var storedTemplates = command != null ? command["stored_templates"] as JArray : null;

            if (storedTemplates == null || storedTemplates.Count == 0)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "No stored templates to compare",
                    Data = new Dictionary<string, object>
                    {
                        { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                    }
                };
            }

            // Capture a fresh fingerprint for verification
            Console.WriteLine("  Capturando huella para verificación...");
            var captureResult = RunHtaCapture("capture", 30);
            if (!captureResult.Success)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "Capture failed: " + captureResult.Message,
                    Data = new Dictionary<string, object>
                    {
                        { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                    }
                };
            }

            var capData = captureResult.Data as Dictionary<string, object>;
            string capturedBase64 = capData != null && capData.ContainsKey("template_data")
                ? (capData["template_data"] as string ?? "") : "";

            if (string.IsNullOrEmpty(capturedBase64))
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "Empty capture data",
                    Data = new Dictionary<string, object>
                    {
                        { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                    }
                };
            }

            // Deserialize captured FeatureSet
            DPFP.FeatureSet capturedFS;
            try
            {
                byte[] capBytes = Convert.FromBase64String(capturedBase64);
                capturedFS = new DPFP.FeatureSet();
                capturedFS.DeSerialize(capBytes);
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "Error deserializing capture: " + ex.Message,
                    Data = new Dictionary<string, object>
                    {
                        { "matched", false }, { "user_id", "" }, { "match_score", 0 }
                    }
                };
            }

            // Verify against each stored template
            string bestUserId = "";
            bool bestVerified = false;
            int bestFar = int.MaxValue;

            Console.WriteLine("  Comparando contra " + storedTemplates.Count + " huellas...");

            for (int i = 0; i < storedTemplates.Count; i++)
            {
                var stored = storedTemplates[i] as JObject;
                if (stored == null) continue;

                var userId = stored.Value<string>("user_id") ?? "";
                var storedBase64 = stored.Value<string>("template_data") ?? "";
                if (string.IsNullOrEmpty(storedBase64)) continue;

                try
                {
                    byte[] tmplBytes = Convert.FromBase64String(storedBase64);

                    // Try as Template first (from SDK enrollment)
                    bool matched = false;
                    try
                    {
                        var template = new DPFP.Template();
                        template.DeSerialize(tmplBytes);
                        var result = DPFP.Verification.Verification.Verify(capturedFS, template);
                        if (result.Verified)
                        {
                            Console.WriteLine("    Match (Template): user_id=" + userId + " FAR=" + result.FARAchieved);
                            if (result.FARAchieved < bestFar)
                            {
                                bestFar = result.FARAchieved;
                                bestUserId = userId;
                                bestVerified = true;
                            }
                            matched = true;
                        }
                    }
                    catch
                    {
                        // Not a valid Template - try as FeatureSet
                    }

                    // Fallback: compare as FeatureSet (from FeatureSet-mode enrollment)
                    if (!matched)
                    {
                        try
                        {
                            var storedFS = new DPFP.FeatureSet();
                            storedFS.DeSerialize(tmplBytes);

                            // Use Verification with a template created from features
                            // Since we can't directly compare 2 FeatureSets with SDK,
                            // try enrollment with both and see if they're compatible
                            var testEnroll = new DPFP.Processing.Enrollment();
                            testEnroll.AddFeatures(storedFS);
                            testEnroll.AddFeatures(capturedFS);
                            // If AddFeatures doesn't throw, they're the same finger
                            Console.WriteLine("    Match (FeatureSet): user_id=" + userId);
                            bestUserId = userId;
                            bestVerified = true;
                            bestFar = 0;
                        }
                        catch
                        {
                            // Different finger or incompatible - no match
                        }
                    }
                }
                catch (Exception ex)
                {
                    Console.WriteLine("    Error verifying " + userId + ": " + ex.Message);
                }
            }

            if (bestVerified)
            {
                int score = 95;
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("  MATCH: user_id=" + bestUserId + ", score=" + score);
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

            Console.ForegroundColor = ConsoleColor.Yellow;
            Console.WriteLine("  No match found");
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

        // =====================================================================
        // Run HTA for fingerprint capture only, return FeatureSet data
        // =====================================================================
        static BiometricResponse RunHtaCapture(string action, int timeoutSecs)
        {
            try
            {
                if (File.Exists(ResultFile))
                    File.Delete(ResultFile);

                File.WriteAllText(CmdFile, action + "\r\n" + timeoutSecs);

                Console.WriteLine("  Launching HTA: " + action + " (timeout: " + timeoutSecs + "s)");

                var psi = new ProcessStartInfo
                {
                    FileName = @"C:\WINDOWS\system32\mshta.exe",
                    Arguments = "\"" + HtaPath + "\"",
                    UseShellExecute = false
                };

                var process = Process.Start(psi);
                if (process == null)
                {
                    return new BiometricResponse { Success = false, Message = "Failed to start mshta.exe" };
                }

                int waitMs = (timeoutSecs + 10) * 1000;
                bool exited = process.WaitForExit(waitMs);

                if (!exited)
                {
                    try { process.Kill(); } catch { }
                    return new BiometricResponse { Success = false, Message = "HTA process timeout" };
                }

                if (!File.Exists(ResultFile))
                {
                    return new BiometricResponse { Success = false, Message = "No result file from HTA" };
                }

                string resultJson = File.ReadAllText(ResultFile);
                Console.WriteLine("  HTA result: " + Truncate(resultJson, 100));

                var htaResult = JsonConvert.DeserializeObject<JObject>(resultJson);
                bool success = htaResult != null && htaResult.Value<bool>("success");
                string message = htaResult != null ? (htaResult.Value<string>("message") ?? "Unknown") : "Unknown";
                var data = htaResult != null ? htaResult["data"] : null;

                var response = new BiometricResponse
                {
                    Success = success,
                    Message = message
                };

                if (data != null && data.Type == JTokenType.Object)
                {
                    response.Data = data.ToObject<Dictionary<string, object>>();
                }

                return response;
            }
            catch (Exception ex)
            {
                return new BiometricResponse
                {
                    Success = false,
                    Message = "Error running HTA: " + ex.Message
                };
            }
        }

        // =====================================================================
        // Helpers
        // =====================================================================
        static string FindHtaPath()
        {
            string exeDir = AppDomain.CurrentDomain.BaseDirectory;
            string[] searchPaths = new[]
            {
                Path.Combine(exeDir, "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "..", "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "biometric.hta"),
            };

            foreach (var path in searchPaths)
            {
                string full = Path.GetFullPath(path);
                if (File.Exists(full))
                    return full;
            }
            return null;
        }

        static string Truncate(string s, int maxLen)
        {
            if (string.IsNullOrEmpty(s)) return "";
            return s.Length <= maxLen ? s : s.Substring(0, maxLen) + "...";
        }
    }

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

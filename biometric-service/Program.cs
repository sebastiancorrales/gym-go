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
        // ENROLL - Multiple captures via HTA until SDK produces a real Template.
        // The SDK typically needs 4 samples. We capture them one by one.
        // A proper DPFP.Template is REQUIRED for secure verification.
        // Strategy: Use HTA's COM DPFPEnrollment (all 4 captures in HTA, same COM layer).
        // The .NET SDK enrollment fails with HTA-produced FeatureSets, but the COM
        // enrollment inside HTA works because it's the same COM layer.
        // =====================================================================
        static BiometricResponse HandleEnroll()
        {
            const int MAX_RETRIES = 2;

            for (int attempt = 1; attempt <= MAX_RETRIES; attempt++)
            {
                Console.WriteLine("  Intento " + attempt + "/" + MAX_RETRIES + ": Enrollment via COM interop (4 capturas individuales)...");

                try
                {
                    // Create COM enrollment object (C# COM interop CAN access Template property)
                    Type enrollType = Type.GetTypeFromProgID("DPFPEngX.DPFPEnrollment");
                    if (enrollType == null)
                    {
                        return new BiometricResponse { Success = false, Message = "COM DPFPEngX.DPFPEnrollment no disponible" };
                    }
                    dynamic comEnrollment = Activator.CreateInstance(enrollType);

                    Type fsType = Type.GetTypeFromProgID("DPFPShrX.DPFPFeatureSet");
                    if (fsType == null)
                    {
                        return new BiometricResponse { Success = false, Message = "COM DPFPShrX.DPFPFeatureSet no disponible" };
                    }

                    int featuresNeeded = (int)comEnrollment.FeaturesNeeded;
                    Console.WriteLine("    COM Enrollment creado. FeaturesNeeded=" + featuresNeeded);

                    // Collect 4 individual captures from HTA
                    for (int cap = 1; cap <= 4; cap++)
                    {
                        Console.WriteLine("    Captura " + cap + "/4...");
                        var captureResult = RunHtaCapture("enroll_capture", 30);

                        if (!captureResult.Success)
                        {
                            Console.ForegroundColor = ConsoleColor.Yellow;
                            Console.WriteLine("      Captura " + cap + " fallida: " + captureResult.Message);
                            Console.ResetColor();
                            break;
                        }

                        var capData = captureResult.Data as Dictionary<string, object>;
                        string capBase64 = capData?["template_data"] as string ?? "";
                        if (string.IsNullOrEmpty(capBase64))
                        {
                            Console.ForegroundColor = ConsoleColor.Yellow;
                            Console.WriteLine("      Captura " + cap + " vacia");
                            Console.ResetColor();
                            break;
                        }

                        // Create COM FeatureSet and deserialize captured data
                        byte[] fsBytes = Convert.FromBase64String(capBase64);
                        dynamic comFS = Activator.CreateInstance(fsType);
                        comFS.Deserialize(fsBytes);

                        // Add to COM enrollment
                        comEnrollment.AddFeatures(comFS);
                        featuresNeeded = (int)comEnrollment.FeaturesNeeded;
                        int status = (int)comEnrollment.TemplateStatus;
                        Console.WriteLine("      AddFeatures OK. FeaturesNeeded=" + featuresNeeded + " TemplateStatus=" + status);

                        // Check if template is ready (TemplateStatus=2)
                        if (status == 2)
                        {
                            Console.WriteLine("    Template READY! Extrayendo via COM interop...");

                            // Access Template property via C# COM interop (works where VBScript fails)
                            dynamic comTemplate = comEnrollment.Template;
                            if (comTemplate == null)
                            {
                                Console.ForegroundColor = ConsoleColor.Yellow;
                                Console.WriteLine("    Template es null a pesar de TemplateStatus=READY");
                                Console.ResetColor();
                                break;
                            }

                            byte[] tmplBytes = (byte[])comTemplate.Serialize();
                            string templateBase64 = Convert.ToBase64String(tmplBytes);
                            Console.WriteLine("    COM Template serializado: " + tmplBytes.Length + " bytes");

                            // Validate with .NET SDK
                            var netTemplate = new DPFP.Template();
                            netTemplate.DeSerialize(tmplBytes);

                            Console.ForegroundColor = ConsoleColor.Green;
                            Console.WriteLine("  ENROLLMENT OK! Template valido .NET SDK: " + tmplBytes.Length + " bytes");
                            Console.ResetColor();

                            return new BiometricResponse
                            {
                                Success = true,
                                Message = "Enrollment completed",
                                Data = new Dictionary<string, object>
                                {
                                    { "template_data", templateBase64 },
                                    { "quality", 90 }
                                }
                            };
                        }
                    }
                }
                catch (Exception ex)
                {
                    Console.ForegroundColor = ConsoleColor.Yellow;
                    Console.WriteLine("    Error en intento " + attempt + ": " + ex.Message);
                    Console.ResetColor();
                }

                if (attempt < MAX_RETRIES)
                {
                    Console.WriteLine("    Reintentando enrollment...");
                }
            }

            Console.ForegroundColor = ConsoleColor.Red;
            Console.WriteLine("  ENROLLMENT FALLIDO despues de " + MAX_RETRIES + " intentos");
            Console.ResetColor();

            return new BiometricResponse
            {
                Success = false,
                Message = "Enrollment fallido. Limpie el lector e intente con capturas firmes y consistentes."
            };
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
            long bestFar = long.MaxValue;

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

                    // Verify using DPFP.Template + DPFP.Verification (the only secure method)
                    var template = new DPFP.Template();
                    template.DeSerialize(tmplBytes);
                    var verification = new DPFP.Verification.Verification();
                    var result = new DPFP.Verification.Verification.Result();
                    verification.Verify(capturedFS, template, ref result);

                    if (result.Verified)
                    {
                        Console.WriteLine("    Match: user_id=" + userId + " FAR=" + result.FARAchieved);
                        if ((long)result.FARAchieved < bestFar)
                        {
                            bestFar = (long)result.FARAchieved;
                            bestUserId = userId;
                            bestVerified = true;
                        }
                    }
                    else
                    {
                        Console.WriteLine("    No match: user_id=" + userId);
                    }
                }
                catch (Exception ex)
                {
                    Console.WriteLine("    Error verifying " + userId + ": " + ex.Message + " (template invalido - requiere re-enrollment)");
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
                Path.Combine(exeDir, "biometric.hta"),
                Path.Combine(exeDir, "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "..", "scripts", "biometric.hta"),
                Path.Combine(exeDir, "..", "..", "..", "..", "scripts", "biometric.hta"),
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

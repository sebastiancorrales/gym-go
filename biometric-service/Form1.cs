using System;
using System.Collections.Generic;
using System.IO;
using System.Net;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Forms;
using DPUruNet;

namespace BiometricPOC
{
    public partial class Form1 : Form
    {
        private ReaderCollection _readers;
        private Reader _reader;

        private enum State { Idle, Enrolling, Matching }
        private volatile State _state = State.Idle;

        // Enrollment
        private List<Fmd> _enrollFmds = new List<Fmd>();
        private TaskCompletionSource<Fmd> _enrollTcs;

        // Match
        private List<MatchTemplate> _matchTemplates;
        private TaskCompletionSource<string> _matchTcs; // devuelve user_id o null

        private volatile bool _running = false;
        private Thread _captureThread;
        private HttpListener _httpListener;

        private const int FALSE_MATCH_RATE = 21474;
        private const int HTTP_PORT = 5001;

        public Form1()
        {
            InitializeComponent();
            InitReader();
            StartHttpServer();
        }

        // -------------------------------------------------------
        // Inicializar lector
        // -------------------------------------------------------
        void InitReader()
        {
            try
            {
                _readers = ReaderCollection.GetReaders();
                if (_readers.Count == 0) { Log("❌ No se detectó ningún lector."); return; }

                _reader = _readers[0];
                Log("✅ Lector: " + _reader.Description.SerialNumber);

                var r = _reader.Open(Constants.CapturePriority.DP_PRIORITY_COOPERATIVE);
                if (r != Constants.ResultCode.DP_SUCCESS) { Log("❌ Error abriendo lector: " + r); return; }

                _reader.GetStatus();
                Log("   Estado: " + _reader.Status.Status);

                _running = true;
                _captureThread = new Thread(CaptureLoop) { IsBackground = true };
                _captureThread.Start();

                Log("🟢 Lector listo.");
            }
            catch (Exception ex) { Log("❌ " + ex.Message); }
        }

        // -------------------------------------------------------
        // Loop de captura persistente
        // -------------------------------------------------------
        void CaptureLoop()
        {
            while (_running)
            {
                if (_state == State.Idle) { Thread.Sleep(100); continue; }

                try
                {
                    CaptureResult result = _reader.Capture(
                        Constants.Formats.Fid.ANSI,
                        Constants.CaptureProcessing.DP_IMG_PROC_DEFAULT,
                        10000,
                        _reader.Capabilities.Resolutions[0]);

                    if (!_running) break;
                    if (_state == State.Idle) continue;
                    if (result == null || result.ResultCode != Constants.ResultCode.DP_SUCCESS) continue;
                    if (result.Quality != Constants.CaptureQuality.DP_QUALITY_GOOD)
                    {
                        Log("⚠️ Calidad baja, intenta de nuevo.");
                        continue;
                    }

                    var format = (_state == State.Enrolling)
                        ? Constants.Formats.Fmd.DP_PRE_REGISTRATION
                        : Constants.Formats.Fmd.DP_VERIFICATION;

                    DataResult<Fmd> fmdResult = FeatureExtraction.CreateFmdFromFid(result.Data, format);
                    if (fmdResult.ResultCode != Constants.ResultCode.DP_SUCCESS)
                    {
                        Log("❌ Error extrayendo FMD: " + fmdResult.ResultCode);
                        continue;
                    }

                    Thread.Sleep(800); // esperar que levante el dedo

                    if (_state == State.Enrolling) ProcessEnroll(fmdResult.Data);
                    else if (_state == State.Matching) ProcessMatch(fmdResult.Data);
                }
                catch (ThreadInterruptedException) { break; }
                catch (Exception ex) { if (_running) Log("❌ " + ex.Message); Thread.Sleep(500); }
            }
        }

        // -------------------------------------------------------
        // Enrollment
        // -------------------------------------------------------
        void ProcessEnroll(Fmd fmd)
        {
            _enrollFmds.Add(fmd);
            Log($"✅ Muestra {_enrollFmds.Count}/4 capturada.");

            if (_enrollFmds.Count < 4)
            {
                Log($"👆 Pon el dedo... ({_enrollFmds.Count + 1}/4)");
                return;
            }

            DataResult<Fmd> enrollResult = Enrollment.CreateEnrollmentFmd(
                Constants.Formats.Fmd.DP_REGISTRATION, _enrollFmds);

            _enrollFmds.Clear();
            _state = State.Idle;

            if (enrollResult.ResultCode != Constants.ResultCode.DP_SUCCESS)
            {
                Log("❌ Enrollment fallido: " + enrollResult.ResultCode);
                _enrollTcs?.TrySetException(new Exception("Enrollment fallido: " + enrollResult.ResultCode));
                return;
            }

            Log("🎉 Enrollment completo.");
            _enrollTcs?.TrySetResult(enrollResult.Data);
        }

        // -------------------------------------------------------
        // Match
        // -------------------------------------------------------
        void ProcessMatch(Fmd liveFmd)
        {
            _state = State.Idle;

            if (_matchTemplates == null || _matchTemplates.Count == 0)
            {
                _matchTcs?.TrySetResult(null);
                return;
            }

            Log($"🔍 Comparando contra {_matchTemplates.Count} template(s)...");

            string bestUserId = null;

            foreach (var t in _matchTemplates)
            {
                try
                {
                    // Deserializar template guardado (base64 → xml → Fmd)
                    byte[] tmplBytes = Convert.FromBase64String(t.Template);
                    string xml = Encoding.UTF8.GetString(tmplBytes);
                    Fmd storedFmd = Fmd.DeserializeXml(xml);

                    IdentifyResult identifyResult = Comparison.Identify(
                        liveFmd, 0, new[] { storedFmd }, FALSE_MATCH_RATE, 1);

                    if (identifyResult.ResultCode == Constants.ResultCode.DP_SUCCESS
                        && identifyResult.Indexes != null
                        && identifyResult.Indexes.Length > 0)
                    {
                        bestUserId = t.UserId;
                        Log($"✅ Match: {t.UserId}");
                        break;
                    }
                }
                catch (Exception ex)
                {
                    Log($"⚠️ Error comparando {t.UserId}: {ex.Message}");
                }
            }

            if (bestUserId == null) Log("❌ No match.");
            _matchTcs?.TrySetResult(bestUserId);
        }

        // -------------------------------------------------------
        // HTTP Server
        // -------------------------------------------------------
        void StartHttpServer()
        {
            _httpListener = new HttpListener();
            _httpListener.Prefixes.Add($"http://localhost:{HTTP_PORT}/");
            _httpListener.Start();
            Log($"🌐 API escuchando en http://localhost:{HTTP_PORT}");

            Task.Run(async () =>
            {
                while (_httpListener.IsListening)
                {
                    try
                    {
                        var ctx = await _httpListener.GetContextAsync();
                        _ = Task.Run(() => HandleRequest(ctx));
                    }
                    catch { }
                }
            });
        }

        async void HandleRequest(HttpListenerContext ctx)
        {
            var req = ctx.Request;
            var res = ctx.Response;
            res.ContentType = "application/json";
            res.Headers.Add("Access-Control-Allow-Origin", "*");

            string path = req.Url.AbsolutePath.ToLower().TrimEnd('/');
            string method = req.HttpMethod.ToUpper();

            try
            {
                string body = "";
                if (req.HasEntityBody)
                    using (var sr = new StreamReader(req.InputStream, req.ContentEncoding))
                        body = await sr.ReadToEndAsync();

                object response;

                if (path == "/status" && method == "GET")
                    response = HandleStatus();
                else if (path == "/enroll" && method == "POST")
                    response = await HandleEnroll();
                else if (path == "/match" && method == "POST")
                    response = await HandleMatch(body);
                else
                {
                    res.StatusCode = 404;
                    response = new { success = false, message = "Endpoint no encontrado" };
                }

                string json = JsonSerializer.Serialize(response);
                byte[] buf = Encoding.UTF8.GetBytes(json);
                res.ContentLength64 = buf.Length;
                await res.OutputStream.WriteAsync(buf, 0, buf.Length);
            }
            catch (Exception ex)
            {
                res.StatusCode = 500;
                byte[] buf = Encoding.UTF8.GetBytes(JsonSerializer.Serialize(new { success = false, message = ex.Message }));
                try { await res.OutputStream.WriteAsync(buf, 0, buf.Length); } catch { }
            }
            finally
            {
                try { res.Close(); } catch { }
            }
        }

        // GET /status
        object HandleStatus()
        {
            bool connected = _reader != null;
            string status = "disconnected";
            if (connected)
            {
                _reader.GetStatus();
                status = _reader.Status.Status.ToString().ToLower();
            }
            Log("[API] GET /status");
            return new { success = true, connected, status, state = _state.ToString().ToLower() };
        }

        // POST /enroll
        // Devuelve template en base64 cuando las 4 capturas están completas
        async Task<object> HandleEnroll()
        {
            if (_reader == null)
                return new { success = false, message = "Lector no conectado" };
            if (_state != State.Idle)
                return new { success = false, message = "Lector ocupado: " + _state };

            Log("[API] POST /enroll — iniciando...");

            _enrollFmds.Clear();
            _enrollTcs = new TaskCompletionSource<Fmd>();
            _state = State.Enrolling;
            Log("👆 Pon el dedo... (1/4)");

            // Esperar máximo 120 segundos
            using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(120));
            cts.Token.Register(() => _enrollTcs.TrySetCanceled());

            try
            {
                Fmd fmd = await _enrollTcs.Task;
                string xml = Fmd.SerializeXml(fmd);
                string base64 = Convert.ToBase64String(Encoding.UTF8.GetBytes(xml));
                return new { success = true, template = base64 };
            }
            catch (TaskCanceledException)
            {
                _state = State.Idle;
                _enrollFmds.Clear();
                return new { success = false, message = "Timeout: el usuario no puso el dedo a tiempo" };
            }
            catch (Exception ex)
            {
                _state = State.Idle;
                return new { success = false, message = ex.Message };
            }
        }

        // POST /match
        // Body: { "templates": [{ "user_id": "xxx", "template": "base64..." }] }
        // Devuelve el user_id que hizo match
        async Task<object> HandleMatch(string body)
        {
            if (_reader == null)
                return new { success = false, message = "Lector no conectado" };
            if (_state != State.Idle)
                return new { success = false, message = "Lector ocupado: " + _state };

            MatchRequest matchReq;
            try { matchReq = JsonSerializer.Deserialize<MatchRequest>(body, new JsonSerializerOptions { PropertyNameCaseInsensitive = true }); }
            catch { return new { success = false, message = "Body JSON inválido" }; }

            if (matchReq?.Templates == null || matchReq.Templates.Count == 0)
                return new { success = false, message = "No se enviaron templates" };

            Log($"[API] POST /match — {matchReq.Templates.Count} template(s)");

            _matchTemplates = matchReq.Templates;
            _matchTcs = new TaskCompletionSource<string>();
            _state = State.Matching;
            Log("👆 Pon el dedo para verificar...");

            using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));
            cts.Token.Register(() => _matchTcs.TrySetCanceled());

            try
            {
                string userId = await _matchTcs.Task;
                if (userId != null)
                    return new { success = true, matched = true, user_id = userId };
                else
                    return new { success = true, matched = false, user_id = (string)null };
            }
            catch (TaskCanceledException)
            {
                _state = State.Idle;
                return new { success = false, message = "Timeout: ninguna huella detectada" };
            }
        }

        // -------------------------------------------------------
        // Botones UI (para pruebas manuales)
        // -------------------------------------------------------
        private void btnEnroll_Click(object sender, EventArgs e)
        {
            if (_reader == null) { Log("❌ No hay lector."); return; }
            _enrollFmds.Clear();
            _enrollTcs = new TaskCompletionSource<Fmd>();
            _state = State.Enrolling;
            Log("--- Enrollment manual ---");
            Log("👆 Pon el dedo... (1/4)");
            _enrollTcs.Task.ContinueWith(t =>
            {
                if (!t.IsFaulted && !t.IsCanceled)
                {
                    string xml = Fmd.SerializeXml(t.Result);
                    string b64 = Convert.ToBase64String(Encoding.UTF8.GetBytes(xml));
                    Log("Template base64 (primeros 60 chars): " + b64.Substring(0, Math.Min(60, b64.Length)) + "...");
                }
            });
        }

        private void btnVerify_Click(object sender, EventArgs e)
            => Log("⚠️ Verificación manual: usa POST /match desde el backend.");

        private void btnClear_Click(object sender, EventArgs e) => txtLog.Clear();

        void Log(string msg)
        {
            if (InvokeRequired) { Invoke(new Action(() => Log(msg))); return; }
            txtLog.AppendText($"[{DateTime.Now:HH:mm:ss}] {msg}\r\n");
            txtLog.ScrollToCaret();
        }

        protected override void OnFormClosing(FormClosingEventArgs e)
        {
            _running = false;
            _state = State.Idle;
            _captureThread?.Interrupt();
            try { _httpListener?.Stop(); } catch { }
            try { _reader?.Dispose(); } catch { }
            try { _readers?.Dispose(); } catch { }
            base.OnFormClosing(e);
        }
    }

    // -------------------------------------------------------
    // DTOs
    // -------------------------------------------------------
    class MatchRequest
    {
        public List<MatchTemplate> Templates { get; set; }
    }
    class MatchTemplate
    {
        public string UserId { get; set; }
        public string Template { get; set; }
    }
}

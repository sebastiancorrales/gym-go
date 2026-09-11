package hikvision

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const maxInt32 = int32(1<<31 - 1)

// captureServer imita al terminal: contesta "deviceError" las primeras
// tantas veces —igual que cuando nadie ha puesto el dedo— y después entrega
// una plantilla.
func captureServer(t *testing.T, emptyAttempts int32) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32

	template := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 512)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if atomic.AddInt32(&calls, 1) <= emptyAttempts {
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><ResponseStatus><statusCode>3</statusCode><statusString>Device Error</statusString><subStatusCode>deviceError</subStatusCode></ResponseStatus>`)
			return
		}
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><CaptureFingerPrint><fingerData>%s</fingerData><fingerNo>7</fingerNo><fingerPrintQuality>82</fingerPrintQuality></CaptureFingerPrint>`, template)
	}))
	return server, &calls
}

// El equipo se rinde solo a los ~11 segundos, que no alcanzan cuando el
// lector está a tres metros del mostrador y fuera de vista. Encadenar
// intentos es lo que le da tiempo al socio de llegar.
func TestCaptureWithinKeepsTryingUntilSomebodyPresentsAFinger(t *testing.T) {
	server, calls := captureServer(t, 2)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	captured, err := client.CaptureFingerprintWithin(7, 10*time.Second)
	if err != nil {
		t.Fatalf("CaptureFingerprintWithin: %v", err)
	}
	if captured.Quality != 82 {
		t.Errorf("calidad = %d, se esperaba 82", captured.Quality)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Errorf("intentos = %d, se esperaban 3 (dos vacíos y el bueno)", got)
	}
}

// Si nadie llega, hay que rendirse y decirlo — no quedarse colgado ni
// reportar un fallo de red, que es como llegaba antes al mostrador.
func TestCaptureWithinGivesUpWhenNobodyComes(t *testing.T) {
	server, calls := captureServer(t, maxInt32)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	start := time.Now()
	_, err := client.CaptureFingerprintWithin(7, 300*time.Millisecond)
	if !errors.Is(err, ErrFingerprintBusy) {
		t.Fatalf("error = %v, se esperaba ErrFingerprintBusy", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("tardó %v en rendirse: la ventana no se está respetando", elapsed)
	}
	if atomic.LoadInt32(calls) < 1 {
		t.Error("no llegó a intentarlo ni una vez")
	}
}

// Una avería real no debe esconderse detrás de 45 segundos de reintentos.
func TestCaptureWithinSurfacesRealFaultsAtOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	start := time.Now()
	if _, err := client.CaptureFingerprintWithin(7, 30*time.Second); err == nil {
		t.Fatal("se esperaba un error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("tardó %v: una avería real debe salir de inmediato", elapsed)
	}
}

// El cliente que espera la huella debe tener más paciencia que el propio
// equipo. Con menos, la respuesta llega justo después del corte y el fallo
// se reporta como problema de red — que fue exactamente lo que pasó en el
// gimnasio: 10s de timeout contra 11s de espera del equipo.
func TestWaitClientOutlastsTheDevicesOwnWait(t *testing.T) {
	client := NewClient("x", 80, "admin", "x")

	if client.waitClient.Timeout <= deviceFingerWait {
		t.Errorf("timeout de captura %v <= espera del equipo %v: la respuesta se corta antes de llegar",
			client.waitClient.Timeout, deviceFingerWait)
	}
	if client.httpClient.Timeout >= client.waitClient.Timeout {
		t.Error("las llamadas normales no deberían esperar tanto como la captura")
	}
}

// Contra el equipo real cada intento tarda ~11s, así que el bucle se pacea
// solo. Si el sensor se averiara y respondiera al instante, sin pausa el
// bucle haría miles de peticiones durante la ventana: un servidor simulado
// llegó a 1000 en 60 ms.
func TestCaptureWithinDoesNotHammerAnInstantlyFailingDevice(t *testing.T) {
	server, calls := captureServer(t, maxInt32)
	defer server.Close()

	client := NewClient("x", 0, "admin", "x")
	client.baseURL = server.URL

	_, _ = client.CaptureFingerprintWithin(7, 1500*time.Millisecond)

	if got := atomic.LoadInt32(calls); got > 6 {
		t.Errorf("se hicieron %d intentos en 1,5s: el bucle está martillando el equipo", got)
	}
}

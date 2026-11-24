package entities

import (
	"strings"
	"testing"
)

func TestNewGiftCardCode_FormatoYAlfabeto(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		code, key, err := NewGiftCardCode("OLI")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(code, "OLI-") || len(code) != 10 {
			t.Fatalf("código %q: se esperaba OLI-XXXXXX", code)
		}
		if key != strings.ReplaceAll(code, "-", "") {
			t.Fatalf("clave %q no es el código sin separador (%q)", key, code)
		}
		// Sin caracteres que se confunden al dictar o teclear.
		if strings.ContainsAny(code[4:], "01OIL") {
			t.Fatalf("código %q usa un carácter ambiguo", code)
		}
		seen[code] = true
	}
	if len(seen) < 495 {
		t.Errorf("sólo %d códigos distintos de 500: el generador no es aleatorio", len(seen))
	}
}

func TestGiftCardCodeKey_NormalizaLoQueSeTeclee(t *testing.T) {
	for _, in := range []string{"OLI-7K3QXM", "oli7k3qxm", " OLI 7K3-QXM ", "oli_7k3qxm"} {
		if got := GiftCardCodeKey(in); got != "OLI7K3QXM" {
			t.Errorf("GiftCardCodeKey(%q) = %q, want OLI7K3QXM", in, got)
		}
	}
}

func TestGiftCard_StateOn_VencidaSeDerivaNoSeGuarda(t *testing.T) {
	card := GiftCard{Status: GiftCardPending, ExpiresOn: "2026-03-10"}
	if s := card.StateOn("2026-03-10"); s != "pending" {
		t.Errorf("el último día todavía se puede redimir: StateOn = %q", s)
	}
	if s := card.StateOn("2026-03-11"); s != GiftCardStateExpired {
		t.Errorf("al día siguiente está vencida: StateOn = %q", s)
	}
	card.Status = GiftCardRedeemed
	if s := card.StateOn("2026-03-11"); s != "redeemed" {
		t.Errorf("una redimida no pasa a vencida: StateOn = %q", s)
	}
}

func TestGiftCardConfig_Defaults(t *testing.T) {
	cfg := GiftCardConfig(nil)
	if cfg.ValidityDays != DefaultGiftCardValidityDays || cfg.CodePrefix != "REG" {
		t.Errorf("sin gimnasio: %+v", cfg)
	}
	cfg = GiftCardConfig(&Gym{Name: "Olimpo Gym"})
	if cfg.CodePrefix != "OLI" {
		t.Errorf("el prefijo por defecto sale del nombre del gimnasio: %q", cfg.CodePrefix)
	}
	cfg = GiftCardConfig(&Gym{Name: "Olimpo", GiftCardSettings: &GiftCardSettings{CodePrefix: "z-9x", ValidityDays: 30}})
	if cfg.CodePrefix != "ZX" || cfg.ValidityDays != 30 {
		t.Errorf("configurado: %+v", cfg)
	}
}

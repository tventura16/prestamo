package service

import (
	"errors"
	"testing"

	"github.com/prestamos/payment-service/internal/repository"
)

func snap(saldo, mora, interes, capital float64) repository.CuotaSnapshot {
	return repository.CuotaSnapshot{
		Capital:        capital,
		Interes:        interes,
		Total:          interes + capital,
		SaldoPendiente: saldo,
		MoraAcumulada:  mora,
		Estado:         "pendiente",
	}
}

func TestDistribuir(t *testing.T) {
	tests := []struct {
		name                    string
		snap                    repository.CuotaSnapshot
		interesYa, capitalYa    float64
		monto                   float64
		wantMora                float64
		wantInteres             float64
		wantCapital             float64
		wantSaldo               float64
		wantNewMora             float64
		wantEstado              string
		wantErr                 bool
	}{
		{
			name:        "pago total exacto",
			snap:        snap(1000, 0, 200, 800),
			monto:       1000,
			wantInteres: 200, wantCapital: 800,
			wantSaldo: 0, wantEstado: "pagada",
		},
		{
			name:     "pago total con mora",
			snap:     snap(1000, 50, 200, 800),
			monto:    1050,
			wantMora: 50, wantInteres: 200, wantCapital: 800,
			wantSaldo: 0, wantNewMora: 0, wantEstado: "pagada",
		},
		{
			name:        "pago parcial solo interes",
			snap:        snap(1000, 0, 200, 800),
			monto:       100,
			wantInteres: 100, wantCapital: 0,
			wantSaldo: 900, wantEstado: "parcial",
		},
		{
			name:     "orden mora primero",
			snap:     snap(500, 30, 100, 400),
			monto:    50,
			wantMora: 30, wantInteres: 20, wantCapital: 0,
			wantSaldo: 480, wantNewMora: 0, wantEstado: "parcial",
		},
		{
			name:      "interes ya pagado va a capital",
			snap:      snap(1000, 0, 200, 800),
			interesYa: 200,
			monto:     300,
			wantInteres: 0, wantCapital: 300,
			wantSaldo: 700, wantEstado: "parcial",
		},
		{
			name:    "overpayment rechazado",
			snap:    snap(1000, 0, 200, 800),
			monto:   1100,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := distribuir(tc.snap, tc.interesYa, tc.capitalYa, tc.monto)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("se esperaba error, got nil")
				}
				if !errors.Is(err, repository.ErrOverpayment) {
					t.Fatalf("se esperaba ErrOverpayment, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if d.Mora != tc.wantMora {
				t.Errorf("mora: got %.2f want %.2f", d.Mora, tc.wantMora)
			}
			if d.Interes != tc.wantInteres {
				t.Errorf("interes: got %.2f want %.2f", d.Interes, tc.wantInteres)
			}
			if d.Capital != tc.wantCapital {
				t.Errorf("capital: got %.2f want %.2f", d.Capital, tc.wantCapital)
			}
			if d.NewSaldo != tc.wantSaldo {
				t.Errorf("newSaldo: got %.2f want %.2f", d.NewSaldo, tc.wantSaldo)
			}
			if d.NewMora != tc.wantNewMora {
				t.Errorf("newMora: got %.2f want %.2f", d.NewMora, tc.wantNewMora)
			}
			if d.Estado != tc.wantEstado {
				t.Errorf("estado: got %s want %s", d.Estado, tc.wantEstado)
			}
		})
	}
}

// TestDistribuirConservaMonto verifica que la suma distribuida nunca exceda
// el monto recibido (invariante financiero).
func TestDistribuirConservaMonto(t *testing.T) {
	montos := []float64{0.01, 1, 99.99, 250.5, 1000}
	s := snap(1000, 50, 200, 800)
	for _, m := range montos {
		d, err := distribuir(s, 0, 0, m)
		if err != nil {
			continue
		}
		suma := d.Mora + d.Interes + d.Capital
		if suma > m+0.005 {
			t.Errorf("monto %.2f: suma distribuida %.2f excede el recibido", m, suma)
		}
	}
}

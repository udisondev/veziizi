package values

import (
	"errors"
	"testing"
)

func TestPayment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payment Payment
		wantErr error
	}{
		{
			name: "valid payment with prepaid terms",
			payment: Payment{
				Price:   &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType: VatTypeIncluded,
				Method:  PaymentMethodBankTransfer,
				Terms:   PaymentTermsPrepaid,
			},
			wantErr: nil,
		},
		{
			name: "valid payment with on_loading terms",
			payment: Payment{
				Price:   &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType: VatTypeIncluded,
				Method:  PaymentMethodBankTransfer,
				Terms:   PaymentTermsOnLoading,
			},
			wantErr: nil,
		},
		{
			name: "valid payment with on_unloading terms",
			payment: Payment{
				Price:   &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType: VatTypeIncluded,
				Method:  PaymentMethodBankTransfer,
				Terms:   PaymentTermsOnUnloading,
			},
			wantErr: nil,
		},
		{
			name: "valid payment with deferred terms and positive days",
			payment: Payment{
				Price:        &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType:      VatTypeIncluded,
				Method:       PaymentMethodBankTransfer,
				Terms:        PaymentTermsDeferred,
				DeferredDays: 30,
			},
			wantErr: nil,
		},
		{
			name: "valid payment with nil price (carriers propose)",
			payment: Payment{
				Price:   nil,
				VatType: VatTypeIncluded,
				Method:  PaymentMethodBankTransfer,
				Terms:   PaymentTermsPrepaid,
			},
			wantErr: nil,
		},
		{
			name: "invalid: deferred terms with zero days",
			payment: Payment{
				Price:        &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType:      VatTypeIncluded,
				Method:       PaymentMethodBankTransfer,
				Terms:        PaymentTermsDeferred,
				DeferredDays: 0,
			},
			wantErr: ErrDeferredDaysMustBePositive,
		},
		{
			name: "invalid: deferred terms with negative days",
			payment: Payment{
				Price:        &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType:      VatTypeIncluded,
				Method:       PaymentMethodBankTransfer,
				Terms:        PaymentTermsDeferred,
				DeferredDays: -5,
			},
			wantErr: ErrDeferredDaysMustBePositive,
		},
		{
			name: "invalid: negative price amount",
			payment: Payment{
				Price:   &Money{Amount: -100, Currency: CurrencyRUB},
				VatType: VatTypeIncluded,
				Method:  PaymentMethodBankTransfer,
				Terms:   PaymentTermsPrepaid,
			},
			wantErr: ErrNegativeAmount,
		},
		{
			name: "deferred days ignored when terms is not deferred",
			payment: Payment{
				Price:        &Money{Amount: 50000, Currency: CurrencyRUB},
				VatType:      VatTypeIncluded,
				Method:       PaymentMethodBankTransfer,
				Terms:        PaymentTermsPrepaid,
				DeferredDays: 0, // should be ignored
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payment.Validate()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() error = %v, wantErr nil", err)
				}
				return
			}

			if err == nil {
				t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

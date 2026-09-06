package main

import "testing"

func TestDecidePaymentAction(t *testing.T) {
	tests := []struct {
		name   string
		event  PaymentEvent
		action string
	}{
		{"ordinary payment", PaymentEvent{ID: "p-1", AmountCents: 2500, Risk: "low"}, "notify"},
		{"high risk payment", PaymentEvent{ID: "p-2", AmountCents: 2500, Risk: "high"}, "review"},
		{"large payment", PaymentEvent{ID: "p-3", AmountCents: 100000, Risk: "low"}, "review"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decide(tt.event).Action; got != tt.action {
				t.Fatalf("action=%q, want %q", got, tt.action)
			}
		})
	}
}

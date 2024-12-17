package report_test

import (
	"accumulation/bandwidth/report"
	"sync"
	"testing"
)

func TestRingBuff(t *testing.T) {
	rb := report.NewRingBuff[int](5)

	// Enqueue
	for i := 0; i < 5; i++ {
		if !rb.Enqueue(i) {
			t.Errorf("Enqueue failed")
		}
	}

	// Dequeue
	for i := 0; i < 5; i++ {
		val, ok := rb.Dequeue()
		if !ok || val != i {
			t.Errorf("Dequeue failed")
		}
		if rb.Size() > 0 && rb.Current() != i+1 {
			t.Errorf("Current failed")
		}
	}

	// Surplus
	for i := 0; i < 5; i++ {
		rb.Enqueue(i)
	}
	surplus := rb.Surplus()
	if len(surplus) != 5 {
		t.Errorf("Surplus failed")
	}

	// SurplusCount
	if rb.Size() != 5 {
		t.Errorf("SurplusCount failed")
	}
}

func TestRingBuffConcurrent(t *testing.T) {
	rb := report.NewRingBuff[int](15000)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		for i := 0; i < 5000; i++ {
			if !rb.Enqueue(i) {
				t.Errorf("Enqueue failed")
			}
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 5000; i++ {
			if !rb.Enqueue(i) {
				t.Errorf("Enqueue failed")
			}
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < 5000; i++ {
			if !rb.Enqueue(i) {
				t.Errorf("Enqueue failed")
			}
		}
		wg.Done()
	}()
	wg.Wait()
	if rb.Size() != 15000 {
		t.Errorf("Enqueue failed")
	}
}

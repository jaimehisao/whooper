package models

import "testing"

func TestFloatPtr(t *testing.T) {
	p := FloatPtr(12.5)
	if p == nil || *p != 12.5 {
		t.Fatalf("FloatPtr = %v", p)
	}
}

func TestRecoveryScoreOrZero(t *testing.T) {
	var nilScore *RecoveryScore
	if nilScore.SpO2OrZero() != 0 || nilScore.SkinTempOrZero() != 0 {
		t.Fatal("nil receiver should return 0")
	}
	s := &RecoveryScore{}
	if s.SpO2OrZero() != 0 || s.SkinTempOrZero() != 0 {
		t.Fatal("nil optional fields should return 0")
	}
	s.SpO2Percentage = FloatPtr(98)
	s.SkinTempCelsius = FloatPtr(33.1)
	if s.SpO2OrZero() != 98 || s.SkinTempOrZero() != 33.1 {
		t.Fatalf("got spo2=%v temp=%v", s.SpO2OrZero(), s.SkinTempOrZero())
	}
}

func TestWorkoutScoreOrZero(t *testing.T) {
	var nilScore *WorkoutScore
	if nilScore.DistanceOrZero() != 0 || nilScore.AltitudeGainOrZero() != 0 || nilScore.AltitudeChangeOrZero() != 0 {
		t.Fatal("nil receiver should return 0")
	}
	s := &WorkoutScore{}
	if s.DistanceOrZero() != 0 {
		t.Fatal("expected 0")
	}
	s.DistanceMeter = FloatPtr(1500)
	s.AltitudeGainMeter = FloatPtr(10)
	s.AltitudeChangeMeter = FloatPtr(-2)
	if s.DistanceOrZero() != 1500 || s.AltitudeGainOrZero() != 10 || s.AltitudeChangeOrZero() != -2 {
		t.Fatalf("got dist=%v gain=%v change=%v", s.DistanceOrZero(), s.AltitudeGainOrZero(), s.AltitudeChangeOrZero())
	}
}

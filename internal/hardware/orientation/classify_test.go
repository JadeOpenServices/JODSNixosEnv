package orientation

import "testing"

func TestIsSensorAcceptsAccelerometer(t *testing.T) {
	if !IsSensor("accel_3d", nil) {
		t.Fatal("accelerometer name was not classified as orientation-capable")
	}
}

func TestIsSensorAcceptsOrientationDevice(t *testing.T) {
	if !IsSensor("relative_orientation", nil) {
		t.Fatal("orientation device was not classified")
	}
}

func TestIsSensorAcceptsAccelerometerChannels(t *testing.T) {
	if !IsSensor("motion", []string{"in_accel_x_raw", "in_accel_y_raw"}) {
		t.Fatal("accelerometer channels were not classified")
	}
}

func TestIsSensorRejectsAmbientLightSensor(t *testing.T) {
	if IsSensor("als", []string{"in_illuminance_raw"}) {
		t.Fatal("ambient-light sensor was misclassified")
	}
}

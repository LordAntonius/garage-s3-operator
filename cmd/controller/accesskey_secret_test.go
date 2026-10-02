package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestSyncAccessKeySecretData_WritesStandardAndLegacyKeys(t *testing.T) {
	secret := &corev1.Secret{}

	if !syncAccessKeySecretData(secret, "GKid", "secret") {
		t.Fatal("expected a new Secret to report a change")
	}

	want := map[string]string{
		"AWS_ACCESS_KEY_ID":     "GKid",
		"AWS_SECRET_ACCESS_KEY": "secret",
		"AWS_ACCESS_KEY":        "GKid",
		"AWS_SECRET_KEY":        "secret",
	}
	for key, value := range want {
		if got := string(secret.Data[key]); got != value {
			t.Errorf("%s = %q, want %q", key, got, value)
		}
	}
}

func TestSyncAccessKeySecretData_AddsStandardKeysToLegacySecret(t *testing.T) {
	secret := &corev1.Secret{Data: map[string][]byte{
		"AWS_ACCESS_KEY": []byte("GKid"),
		"AWS_SECRET_KEY": []byte("secret"),
	}}

	if !syncAccessKeySecretData(secret, "GKid", "secret") {
		t.Fatal("expected a Secret without the standard keys to report a change")
	}
	if got := string(secret.Data["AWS_ACCESS_KEY_ID"]); got != "GKid" {
		t.Errorf("AWS_ACCESS_KEY_ID = %q, want %q", got, "GKid")
	}
	if got := string(secret.Data["AWS_SECRET_ACCESS_KEY"]); got != "secret" {
		t.Errorf("AWS_SECRET_ACCESS_KEY = %q, want %q", got, "secret")
	}
}

func TestSyncAccessKeySecretData_ReportsNoChangeWhenCurrent(t *testing.T) {
	secret := &corev1.Secret{}
	syncAccessKeySecretData(secret, "GKid", "secret")

	if syncAccessKeySecretData(secret, "GKid", "secret") {
		t.Fatal("expected an up-to-date Secret to report no change")
	}
}

func TestSyncAccessKeySecretData_UpdatesRotatedKey(t *testing.T) {
	secret := &corev1.Secret{}
	syncAccessKeySecretData(secret, "GKid", "old")

	if !syncAccessKeySecretData(secret, "GKid", "new") {
		t.Fatal("expected a rotated secret key to report a change")
	}
	for _, key := range []string{"AWS_SECRET_ACCESS_KEY", "AWS_SECRET_KEY"} {
		if got := string(secret.Data[key]); got != "new" {
			t.Errorf("%s = %q, want %q", key, got, "new")
		}
	}
}

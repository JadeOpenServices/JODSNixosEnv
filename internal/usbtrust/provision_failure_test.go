package usbtrust

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type provisionTestSigner struct {
	signErr, verifyErr     error
	onSign                 func()
	signCalls, verifyCalls int
}

func (s *provisionTestSigner) Sign(context.Context, []byte) (Signature, error) {
	s.signCalls++
	if s.onSign != nil {
		s.onSign()
	}
	return Signature{}, s.signErr
}

func (s *provisionTestSigner) Verify(context.Context, []byte, Signature) error {
	s.verifyCalls++
	return s.verifyErr
}

func TestTPMProvisionRollbackOnlyNewUnverifiedKeys(t *testing.T) {
	signFailure := errors.New("signing failed")
	verifyFailure := errors.New("verification failed")
	persistFailure := errors.New("persistent handle became occupied")
	rollbackFailure := errors.New("TPM unavailable during rollback")
	for _, test := range []struct {
		name                    string
		occupied, cancelOnSign  bool
		signErr, verifyErr      error
		persistErr, rollbackErr error
		wantCalls               []string
		wantSign, wantVerify    int
		wantPersistent          bool
		wantErr                 error
	}{
		{
			name: "occupied handle untouched", occupied: true,
			wantCalls: []string{"inspect"}, wantPersistent: true,
		},
		{
			name: "failed persistence never evicts", persistErr: persistFailure,
			wantCalls: []string{"inspect", "create", "persist", "flush"}, wantErr: persistFailure,
			wantPersistent: true,
		},
		{
			name: "sign failure removes new key", signErr: signFailure,
			wantCalls: []string{"inspect", "create", "persist", "rollback", "flush"},
			wantSign:  1, wantErr: signFailure,
		},
		{
			name: "verification failure removes new key", verifyErr: verifyFailure,
			wantCalls: []string{"inspect", "create", "persist", "rollback", "flush"},
			wantSign:  1, wantVerify: 1, wantErr: verifyFailure,
		},
		{
			name: "canceled caller still cleans up", signErr: context.Canceled, cancelOnSign: true,
			wantCalls: []string{"inspect", "create", "persist", "rollback", "flush"},
			wantSign:  1, wantErr: context.Canceled,
		},
		{
			name: "rollback failure is reported", signErr: signFailure, rollbackErr: rollbackFailure,
			wantCalls: []string{"inspect", "create", "persist", "rollback", "flush"},
			wantSign:  1, wantPersistent: true, wantErr: signFailure,
		},
		{
			name:      "verified key stays persistent",
			wantCalls: []string{"inspect", "create", "persist", "flush"},
			wantSign:  1, wantVerify: 1, wantPersistent: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signer := &provisionTestSigner{signErr: test.signErr, verifyErr: test.verifyErr}
			if test.cancelOnSign {
				signer.onSign = cancel
			}
			const handle = "0x81000042"
			persistent := test.occupied
			var calls []string
			command := func(commandCtx context.Context, name string, args ...string) ([]byte, error) {
				if err := commandCtx.Err(); err != nil {
					t.Fatalf("%s received canceled context: %v", name, err)
				}
				switch name {
				case "tpm2_getcap":
					calls = append(calls, "inspect")
					if persistent {
						return []byte("- " + handle + "\n"), nil
					}
				case "tpm2_createprimary":
					calls = append(calls, "create")
				case "tpm2_evictcontrol":
					if len(args) == 6 {
						calls = append(calls, "persist")
						if args[5] != handle {
							t.Fatalf("persisted at wrong handle: %v", args)
						}
						// A competing owner can occupy the previously unused handle
						// before persistence; failure must not evict its object.
						persistent = true
						return nil, test.persistErr
					}
					calls = append(calls, "rollback")
					if !reflect.DeepEqual(args, []string{"-Q", "-C", "o", "-c", handle}) {
						t.Fatalf("unexpected rollback target: %v", args)
					}
					if _, ok := commandCtx.Deadline(); !ok {
						t.Fatal("rollback has no timeout")
					}
					if test.rollbackErr != nil {
						return []byte("simulated TPM failure"), test.rollbackErr
					}
					persistent = false
				case "tpm2_flushcontext":
					calls = append(calls, "flush")
				default:
					t.Fatalf("unexpected TPM command: %s %v", name, args)
				}
				return nil, nil
			}
			err := provisionTPM(ctx, t.TempDir(), handle, signer, command)
			if test.occupied {
				if err == nil || !strings.Contains(err.Error(), "occupied") {
					t.Fatalf("occupied handle was not refused: %v", err)
				}
			} else if !errors.Is(err, test.wantErr) {
				t.Fatalf("error=%v, want %v", err, test.wantErr)
			}
			if test.rollbackErr != nil && (!errors.Is(err, test.rollbackErr) ||
				!strings.Contains(err.Error(), "handle may remain occupied")) {
				t.Fatalf("rollback failure was hidden: %v", err)
			}
			if !reflect.DeepEqual(calls, test.wantCalls) {
				t.Fatalf("commands=%v, want %v", calls, test.wantCalls)
			}
			if persistent != test.wantPersistent {
				t.Fatalf("persistent=%t, want %t", persistent, test.wantPersistent)
			}
			if signer.signCalls != test.wantSign || signer.verifyCalls != test.wantVerify {
				t.Fatalf("self-test calls: sign=%d verify=%d", signer.signCalls, signer.verifyCalls)
			}
		})
	}
}

func TestTPMProvisionRefusesExistingStateBeforeTPMAccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, EnvelopeFile), []byte("existing signed state"), 0600); err != nil {
		t.Fatal(err)
	}
	command := func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("existing state reached TPM provisioning")
		return nil, nil
	}
	err := provisionTPM(context.Background(), dir, "0x81000042", &provisionTestSigner{}, command)
	if err == nil || !strings.Contains(err.Error(), "existing USB trust state") {
		t.Fatalf("existing state was not refused: %v", err)
	}
}

# Security

SOPS/age protects service secrets. Last.fm and ListenBrainz credentials are
entered in the terminal and encrypted; never put tokens in JSON.

The LUKS flow supports existing-key retention, TPM2 unlock enrollment,
confirmed key rotation, and a separate displayed recovery key. Recovery keys
are not written to repository files.

The optional work account is `<username>-corp` with a separate home directory
and reduced application set.

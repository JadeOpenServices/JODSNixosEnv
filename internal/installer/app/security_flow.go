package app

func mayOfferSecureBootFallback(
	secureBootEnabled bool,
	managed bool,
	unattended bool,
) bool {
	return secureBootEnabled && !managed && !unattended
}

func tpm2AllowedForSecureBoot(secureBootEnabled bool) bool {
	return secureBootEnabled
}

// tpm2FollowsRequest reports whether the rendered TPM2 unlock setting is the
// user's request instead of a probe of the running system. Managed endpoints
// are TPM2-bound by contract, and a fresh install's LUKS2 root is created by
// rootprovision after this decision, so the live media's mappings say nothing
// about it (e2e-target, 2026-09-29: rendered false, TPM2 never enrolled).
func tpm2FollowsRequest(endpointManaged, persistentInstalledHost bool) bool {
	return endpointManaged || !persistentInstalledHost
}

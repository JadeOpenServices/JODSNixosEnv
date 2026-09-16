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

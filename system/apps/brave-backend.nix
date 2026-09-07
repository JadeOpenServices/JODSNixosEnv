{
  lib,
  settings,
  ...
}:

let
  projectWebAppsEnabled =
    (settings.planeEnable or false)
    || (settings.drawioEnable or false);
in
{
  config = lib.mkIf projectWebAppsEnabled {
    environment.etc."brave/policies/managed/gjallaros.json".text =
      builtins.toJSON {
        # ----------------------------------------------------
        # Brave services / promotional features
        # ----------------------------------------------------
        BraveRewardsDisabled = true;
        BraveWalletDisabled = true;
        BraveVPNDisabled = true;
        BraveAIChatEnabled = false;
        BraveNewsDisabled = true;
        BraveTalkDisabled = true;
        TorDisabled = true;

        # ----------------------------------------------------
        # Brave analytics / telemetry
        # ----------------------------------------------------
        BraveP3AEnabled = false;
        BraveStatsPingEnabled = false;
        BraveWebDiscoveryEnabled = false;

        MetricsReportingEnabled = false;
        UrlKeyedAnonymizedDataCollectionEnabled = false;
        UserFeedbackAllowed = false;
        WebRtcEventLogCollectionAllowed = false;

        # ----------------------------------------------------
        # Privacy protections
        # ----------------------------------------------------
        BraveGlobalPrivacyControlEnabled = true;
        BraveDeAmpEnabled = true;
        BraveDebouncingEnabled = true;
        BraveTrackingQueryParametersFilteringEnabled = true;
        BraveReduceLanguageEnabled = true;

        # Block third-party cookies but retain first-party
        # cookies so Plane/Draw.io sessions remain persistent.
        BlockThirdPartyCookies = true;

        # ----------------------------------------------------
        # Malware / phishing protection
        #
        # Keep Standard Safe Browsing. Enhanced protection
        # would trade additional browsing data for protection.
        # ----------------------------------------------------
        SafeBrowsingProtectionLevel = 1;
        SafeBrowsingExtendedReportingEnabled = false;

        # ----------------------------------------------------
        # Backend-browser behaviour
        # ----------------------------------------------------
        SyncDisabled = true;
        BackgroundModeEnabled = false;
        DefaultBrowserSettingEnabled = false;

        # This backend exists for web apps, not Chromium
        # account/payment/profile conveniences.
        AutofillAddressEnabled = false;
        AutofillCreditCardEnabled = false;
        PaymentMethodQueryEnabled = false;
        PromotionalTabsEnabled = false;
      };
  };
}

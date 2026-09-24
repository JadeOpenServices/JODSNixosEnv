{
  lib,
  settings,
  ...
}:

let
  canonicalWebApplications = settings ? webApplications;
  webApplications = settings.webApplications or [ ];

  genericBraveWebApplicationEnabled = builtins.any (
    application:
    let
      id = application.id or "";
    in
    id == "plane" || id == "drawio"
  ) webApplications;

  legacyBraveWebApplicationEnabled =
    (settings.planeEnable or false)
    || (settings.drawioEnable or false);

  projectWebAppsEnabled =
    if canonicalWebApplications
    then genericBraveWebApplicationEnabled
    else legacyBraveWebApplicationEnabled;
in
{
  config = lib.mkIf projectWebAppsEnabled {
    environment.etc."brave/policies/managed/gjallaros.json".text =
      builtins.toJSON {
        BraveRewardsDisabled = true;
        BraveWalletDisabled = true;
        BraveVPNDisabled = true;
        BraveAIChatEnabled = false;
        BraveNewsDisabled = true;
        BraveTalkDisabled = true;
        TorDisabled = true;

        BraveP3AEnabled = false;
        BraveStatsPingEnabled = false;
        BraveWebDiscoveryEnabled = false;

        MetricsReportingEnabled = false;
        UrlKeyedAnonymizedDataCollectionEnabled = false;
        UserFeedbackAllowed = false;
        WebRtcEventLogCollectionAllowed = false;

        BraveGlobalPrivacyControlEnabled = true;
        BraveDeAmpEnabled = true;
        BraveDebouncingEnabled = true;
        BraveTrackingQueryParametersFilteringEnabled = true;
        BraveReduceLanguageEnabled = true;

        BlockThirdPartyCookies = true;

        SafeBrowsingProtectionLevel = 1;
        SafeBrowsingExtendedReportingEnabled = false;

        SyncDisabled = true;
        BackgroundModeEnabled = false;
        DefaultBrowserSettingEnabled = false;

        AutofillAddressEnabled = false;
        AutofillCreditCardEnabled = false;
        PaymentMethodQueryEnabled = false;
        PromotionalTabsEnabled = false;
      };
  };
}

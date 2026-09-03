{
  config,
  lib,
  settings,
  ...
}:
{
  services.mpdscribble = lib.mkIf settings.enableScrobbling {
    enable = true;
    verbose = 3;
    endpoints = {
      "last.fm" = lib.mkIf settings.enableLastfm {
        passwordFile = "/run/secrets/lastfm";
        username = settings.lastfmUsername;
      };
      "listenbrainz" = lib.mkIf settings.enableListenbrainz {
        passwordFile = "/run/secrets/listenbrainz";
        username = settings.listenbrainzUsername;
      };
    };
  };
}

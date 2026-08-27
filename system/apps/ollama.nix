{ lib, settings, pkgs, ... }:

let
  ollamaPackage = if settings.graphicsVendor == "amd" && builtins.hasAttr "ollama-rocm" pkgs
    then pkgs.ollama-rocm else pkgs.ollama;
  ollamaEnvironment = {
    OLLAMA_NUM_PARALLEL = "1";
    OLLAMA_MAX_LOADED_MODELS = "1";
    OLLAMA_KEEP_ALIVE = "5m";
    OLLAMA_CONTEXT_LENGTH = toString settings.aiContextTokens;
    OLLAMA_FLASH_ATTENTION = "1";
    OLLAMA_KV_CACHE_TYPE = "q8_0";
  };
in

{
    environment.systemPackages = with pkgs; [
        ollamaPackage
        (writeShellScriptBin "gjallar-ai" ''
          exec ${ollamaPackage}/bin/ollama run ${lib.escapeShellArg settings.aiModel} "$@"
        '')
    ];

    services.ollama = rec {
        enable = true;

        # Keep the model API local; expose it deliberately through a reverse
        # proxy or VPN if remote access is ever required.
        openFirewall = false;
        host = "127.0.0.1";
        port = 11434;
        environmentVariables = ollamaEnvironment;

        package = ollamaPackage;
    };

    systemd.services.ollama.serviceConfig = {
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = "read-only";
        ReadWritePaths = [ "/var/lib/ollama" "/var/cache/ollama" ];
        RestrictAddressFamilies = [ "AF_UNIX" "AF_INET" "AF_INET6" ];
    };
}

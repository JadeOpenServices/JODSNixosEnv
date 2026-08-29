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

lib.mkIf (if settings ? aiEnable then settings.aiEnable else false) {
    environment.systemPackages = with pkgs; [
        ollamaPackage
        (writeShellScriptBin "gjallar-ai" ''
          exec ${ollamaPackage}/bin/ollama run ${lib.escapeShellArg settings.aiModel} \
            --system "Caveman mode: use few words, keep meaning. Lead with the result; no greetings, filler, repetition, or long background. Use compact bullets when useful. Keep code, commands, paths, errors, identifiers, and safety caveats exact. Expand only when asked or safety requires it." "$@"
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

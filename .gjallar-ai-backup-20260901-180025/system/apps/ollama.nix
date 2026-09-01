{ lib, settings, pkgs, ... }:

let
  aiSystemPrompt = (builtins.fromJSON (builtins.readFile ./ai/system-prompt.json)).system;

  ollamaPackage =
    if settings.graphicsVendor == "amd"
       && builtins.hasAttr "ollama-rocm" pkgs
    then pkgs.ollama-rocm
    else pkgs.ollama;

  # Ollama normally ignores integrated GPUs. AMD APUs such as the
  # Radeon 780M can nevertheless be useful for local inference, so
  # explicitly enable Ollama's iGPU support when the installer detected
  # an AMD integrated graphics controller.
  #
  # This is deliberately derived from installer-generated settings rather
  # than hard-coded to a particular laptop/GPU model.
  ollamaIgpuEnable =
    lib.optionalAttrs
      (
        (if settings ? aiEnable then settings.aiEnable else false)
        && settings.graphicsVendor == "amd"
        && settings.graphicsType == "integrated"
        && settings.graphicsIntegratedBusId != ""
      )
      {
        OLLAMA_IGPU_ENABLE = "1";
      };

  ollamaEnvironment = {
    OLLAMA_NUM_PARALLEL = "1";
    OLLAMA_MAX_LOADED_MODELS = "1";
    OLLAMA_KEEP_ALIVE = "5m";
    OLLAMA_CONTEXT_LENGTH = toString settings.aiContextTokens;
    OLLAMA_FLASH_ATTENTION = "1";
    OLLAMA_KV_CACHE_TYPE = "q8_0";
  } // ollamaIgpuEnable;

  securityInstructions = builtins.fromJSON (builtins.readFile ./ai/security-instructions.json);

  aiCli = pkgs.writeShellScriptBin "gjallar-ai" ''
    set -eu

    assistant_model="gjallaros-caveman-ai"
    modelfile="$(mktemp)"
    trap 'rm -f "$modelfile"' EXIT

    cat > "$modelfile" <<EOF
FROM ${settings.aiModel}
SYSTEM """${securityInstructions.system}"""
EOF

    ${ollamaPackage}/bin/ollama create "$assistant_model" --file "$modelfile"

    exec ${ollamaPackage}/bin/ollama run "$assistant_model" "$@"
  '';

  aiLauncher = pkgs.makeDesktopItem {
    name = "gjallarOS-ai";
    desktopName = "gjallarOS-ai";
    genericName = "Local AI assistant";
    comment = "Private local AI assistant powered by Ollama";
    exec =
      "${pkgs.kitty}/bin/kitty --title gjallarOS-ai --class gjallarOS-ai -- ${aiCli}/bin/gjallar-ai";
    icon =
      "${pkgs.papirus-icon-theme}/share/icons/Papirus/64x64/apps/devassistant.svg";
    categories = [ "Utility" "Development" "Chat" ];
    startupNotify = true;
  };
in

lib.mkIf (if settings ? aiEnable then settings.aiEnable else false) {
  environment.systemPackages = with pkgs; [
    ollamaPackage
    aiCli
    aiLauncher
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
    ProtectHome = lib.mkForce "read-only";

    ReadWritePaths = [
      "/var/lib/ollama"
    ];

    RestrictAddressFamilies = [
      "AF_UNIX"
      "AF_INET"
      "AF_INET6"
    ];
  };
}

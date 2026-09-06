{
  lib,
  settings,
  pkgs,
  ...
}:

let
  aiSystemPrompt = (builtins.fromJSON (builtins.readFile ./ai/system-prompt.json)).system;

  ollamaPackage =
    if settings.graphicsVendor == "amd" && builtins.hasAttr "ollama-rocm" pkgs then
      pkgs.ollama-rocm
    else
      pkgs.ollama;

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
  }
  // ollamaIgpuEnable;

  securityInstructions = builtins.fromJSON (builtins.readFile ./ai/security-instructions.json);

  assistantModel = "gjallaros-caveman-ai";
  modelDefinition = ''
    FROM ${settings.aiModel}
    ${lib.optionalString (accelerationProfile.numGpu != null) "PARAMETER num_gpu ${toString accelerationProfile.numGpu}"}
    SYSTEM """${aiSystemPrompt}

    ${securityInstructions.system}"""
    PARAMETER num_ctx ${toString settings.aiContextTokens}
  '';
  # Ollama acceleration policy.
  #
  # "auto" preserves Ollama's normal automatic GPU placement.
  # "integrated-safe" deliberately offloads only a few layers so the
  # desktop-driving iGPU is never given the entire model at once.
  # "cpu-safe" disables model-layer GPU offload completely.
  #
  # Keep this selector on "auto" until a profile is explicitly validated.
  accelerationProfileName = "integrated-safe";

  accelerationProfiles = {
    auto = {
      numGpu = null;
      description = "Ollama automatic GPU placement";
    };

    integrated-safe = {
      numGpu = 4;
      description = "Conservative partial offload for a desktop-driving integrated GPU";
    };

    cpu-safe = {
      numGpu = 0;
      description = "CPU-only model execution";
    };
  };

  accelerationProfile =
    accelerationProfiles.${accelerationProfileName}
      or (throw "Unknown Ollama acceleration profile: ${accelerationProfileName}");

  modelDefinitionFile = pkgs.writeText "gjallaros-caveman-ai.Modelfile" modelDefinition;
  modelDefinitionHash = builtins.hashString "sha256" modelDefinition;

  agentMode =
    if settings ? aiAgentMode
    then settings.aiAgentMode
    else "workspace";

  managedOpencodeConfig =
    pkgs.writeText "gjallar-opencode-managed.json" ''
      {
        "$schema": "https://opencode.ai/config.json",

        "autoupdate": false,
        "share": "disabled",

        "provider": {
          "ollama": {
            "npm": "@ai-sdk/openai-compatible",
            "name": "GjallarOS Model Broker",

            "options": {
              "baseURL": "http://127.0.0.1:11434/v1"
            },

            "models": {
              "${assistantModel}": {
                "name": "GjallarOS Local ${assistantModel}"
              }
            }
          }
        },

        "model": "ollama/${assistantModel}",

        "permissions": [
          {
            "action": "*",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "read",
            "resource": "*",
            "effect": "allow"
          },

          {
            "action": "glob",
            "resource": "*",
            "effect": "allow"
          },

          {
            "action": "grep",
            "resource": "*",
            "effect": "allow"
          },

          {
            "action": "list",
            "resource": "*",
            "effect": "allow"
          },

          {
            "action": "edit",
            "resource": "*",
            "effect": "allow"
          },

          {
            "action": "read",
            "resource": "*.env",
            "effect": "deny"
          },

          {
            "action": "read",
            "resource": "*.env.*",
            "effect": "deny"
          },

          {
            "action": "read",
            "resource": "**/.ssh/**",
            "effect": "deny"
          },

          {
            "action": "read",
            "resource": "**/.gnupg/**",
            "effect": "deny"
          },

          {
            "action": "read",
            "resource": "**/.password-store/**",
            "effect": "deny"
          },

          {
            "action": "external_directory",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "webfetch",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "websearch",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "subagent",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "shell",
            "resource": "*",
            "effect": "deny"
          },

          {
            "action": "shell",
            "resource": "gjallar-agent-tool *",
            "effect": "allow"
          },

          {
            "action": "shell",
            "resource": "gjallar-research *",
            "effect": "allow"
          }
        ],

        "agent": {
          "gjallar": {
            "description": "GjallarOS local engineering agent",
            "mode": "primary",
            "model": "ollama/${assistantModel}",
            "steps": 100,

            "permission": {
              "shell": {
                "*": "deny",
                "gjallar-agent-tool *": "allow",
                "gjallar-research *": "allow"
              },

              "webfetch": "deny",
              "websearch": "deny",
              "external_directory": "deny",
              "task": "deny"
            }
          }
        }
      }
    '';

  aiSessionRuntime =
    pkgs.writeShellScript "gjallar-ai-session-runtime" ''
      set -euo pipefail
      umask 077

      trace_file="$RUNTIME_DIRECTORY/startup.log"

      exec 3>>"$trace_file"

      ${pkgs.coreutils}/bin/chmod 0644 "$trace_file"

      trap '
        rc=$?
        printf "FAIL rc=%s line=%s command=%q\\n" \
          "$rc" "$LINENO" "$BASH_COMMAND" >&3
      ' ERR

      printf "BEGIN pid=%s instance=%q\\n" "$$" "''${1:-}" >&3

      die() {
        printf 'gjallar-ai-session: %s\n' "$*" >&2
        exit 1
      }

      instance="''${1:-}"

      [ -n "$instance" ] ||
        die "missing systemd instance"

      username=${lib.escapeShellArg settings.username}

      uid="$(
        ${pkgs.coreutils}/bin/id -u "$username"
      )"

      gid="$(
        ${pkgs.coreutils}/bin/id -g "$username"
      )"

      # PID 1 has already bind-mounted the path encoded by this
      # template instance onto /workspace.
      workspace="/workspace"

      [ -d "$workspace" ] ||
        die "systemd workspace bind is missing"

      top="$(
        ${pkgs.util-linux}/bin/runuser \
          -u "$username" \
          -- \
          ${pkgs.git}/bin/git \
            -C "$workspace" \
            rev-parse \
            --show-toplevel
      )" ||
        die "workspace is not a Git repository"

      top="$(
        ${pkgs.coreutils}/bin/realpath \
          -e \
          -- \
          "$top"
      )"

      [ "$top" = "$workspace" ] ||
        die "workspace must be the Git repository root"

      for forbidden in \
        /workspace/opencode.json \
        /workspace/opencode.jsonc \
        /workspace/.opencode
      do
        if [ -e "$forbidden" ] || [ -L "$forbidden" ]; then
          die \
            "project-local OpenCode configuration is forbidden: $forbidden"
        fi
      done

      ${pkgs.coreutils}/bin/install \
        -d \
        -m 0700 \
        -o "$username" \
        -g "$gid" \
        "$RUNTIME_DIRECTORY/home" \
        "$RUNTIME_DIRECTORY/config" \
        "$RUNTIME_DIRECTORY/cache" \
        "$RUNTIME_DIRECTORY/data" \
        "$RUNTIME_DIRECTORY/state"

      # PID 1 created RuntimeDirectory for the root-owned service. The
      # console listener later runs as the desktop UID, so hand ownership
      # of this per-session directory to that UID before privilege drop.
      ${pkgs.coreutils}/bin/chown \
        "$uid:$gid" \
        "$RUNTIME_DIRECTORY"

      ${pkgs.coreutils}/bin/chmod \
        0700 \
        "$RUNTIME_DIRECTORY"

      ${pkgs.util-linux}/bin/runuser \
        -u "$username" \
        -- \
        ${pkgs.socat}/bin/socat \
          TCP-LISTEN:11434,bind=127.0.0.1,reuseaddr,fork \
          UNIX-CONNECT:/run/gjallar-ai-model/model.sock &

      bridge_pid=$!

      cleanup() {
        kill "$bridge_pid" 2>/dev/null || true
        wait "$bridge_pid" 2>/dev/null || true
      }

      trap cleanup EXIT INT TERM HUP

      ready=false

      for _ in $(${pkgs.coreutils}/bin/seq 1 50); do
        if ${pkgs.curl}/bin/curl \
          --silent \
          --fail \
          --max-time 1 \
          http://127.0.0.1:11434/health \
          >/dev/null 2>&1
        then
          ready=true
          break
        fi

        ${pkgs.coreutils}/bin/sleep 0.1
      done

      [ "$ready" = true ] ||
        die "model broker bridge did not become ready"

      export HOME="$RUNTIME_DIRECTORY/home"
      export XDG_CONFIG_HOME="$RUNTIME_DIRECTORY/config"
      export XDG_CACHE_HOME="$RUNTIME_DIRECTORY/cache"
      export XDG_DATA_HOME="$RUNTIME_DIRECTORY/data"
      export XDG_STATE_HOME="$RUNTIME_DIRECTORY/state"

      export OPENAI_API_KEY=gjallar-local
      export OPENAI_BASE_URL=http://127.0.0.1:11434/v1

      unset \
        ANTHROPIC_API_KEY \
        ANTHROPIC_AUTH_TOKEN \
        GOOGLE_API_KEY \
        OPENROUTER_API_KEY

      export PATH="/etc/profiles/per-user/$username/bin:/run/current-system/sw/bin"

      exec ${pkgs.util-linux}/bin/setpriv \
        --reuid="$uid" \
        --regid="$gid" \
        --init-groups \
        --bounding-set=-all \
        --inh-caps=-all \
        --ambient-caps=-all \
        -- \
        ${pkgs.socat}/bin/socat \
          UNIX-LISTEN:"$RUNTIME_DIRECTORY/console.sock",mode=0600 \
          EXEC:${lib.escapeShellArg "${pkgs.opencode}/bin/opencode --cwd /workspace"},pty,setsid,ctty,stderr
    '';

  aiDiagnostics = pkgs.writeShellScriptBin "gjallar-ai-diagnostics" ''
    set -u
    echo "AI enabled: true"
    echo "Selection: resolved during installation"
    echo "Resolved base model: ${settings.aiModel}"
    echo "Context tokens: ${toString settings.aiContextTokens}"
    echo "Detected VRAM MiB: ${toString settings.aiVramMB}"
    echo "Agent mode: ${if settings ? aiAgentMode then settings.aiAgentMode else "workspace"}"
    echo "Derived model: ${assistantModel}"
    echo "Definition hash: ${modelDefinitionHash}"
    printf 'Ollama service: '; ${pkgs.systemd}/bin/systemctl is-active ollama.service 2>/dev/null || true
    printf 'Provision service: '; ${pkgs.systemd}/bin/systemctl is-active ollama-model-provision.service 2>/dev/null || true
    printf 'Model broker: '; ${pkgs.systemd}/bin/systemctl is-active gjallar-ai-model.service 2>/dev/null || true
    printf 'Research broker: '; ${pkgs.systemd}/bin/systemctl is-active gjallar-ai-research.service 2>/dev/null || true
    printf 'Model broker health: '
    if ${pkgs.curl}/bin/curl \
      --silent \
      --fail \
      --max-time 3 \
      --unix-socket /run/gjallar-ai-model/model.sock \
      http://localhost/health >/dev/null
    then
      echo reachable
    else
      echo unavailable
    fi
    state=/var/lib/gjallaros-agent/definition.sha256
    if [ -r "$state" ]; then printf 'Applied hash: '; ${pkgs.coreutils}/bin/head -c 128 "$state"; echo; else echo "Applied hash: unavailable"; fi
    if command -v opencode-local >/dev/null 2>&1; then echo "opencode-local: available"; else echo "opencode-local: unavailable"; fi
  '';
in

lib.mkIf (if settings ? aiEnable then settings.aiEnable else false) {
  environment.systemPackages = with pkgs; [
    aiDiagnostics
  ];


  assertions = [
    {
      assertion = agentMode == "workspace";

      message =
        "The hardened GjallarOS AI session currently requires "
        + "aiAgentMode = \"workspace\".";
    }
  ];

  environment.etc."opencode/opencode.json".source =
    managedOpencodeConfig;

  users.groups.gjallar-ai-model = { };

  users.users.gjallar-ai-model = {
    isSystemUser = true;
    group = "gjallar-ai-model";
  };

  users.users.${settings.username}.extraGroups = lib.mkAfter [
    "gjallar-ai-model"
  ];

  systemd.tmpfiles.rules = [
    "d /workspace 0755 root root -"

    # Ollama previously used DynamicUser and therefore may leave its
    # StateDirectory tree owned by the old transient UID. Preserve file
    # modes but recursively transfer ownership to the stable service account.
    "d /var/lib/ollama 0750 ollama ollama -"
    "Z /var/lib/ollama - ollama ollama -"
  ];

  # Keep the machine's existing firewall backend unchanged. This isolated
  # nftables table exists only to enforce the local Ollama transport boundary.
  #
  # Ollama must never start unless this gate was installed successfully.
  systemd.services.gjallar-ai-model-firewall = {
    description = "GjallarOS local Ollama API firewall";

    before = [
      "ollama.service"
      "gjallar-ai-model.service"
      "ollama-model-provision.service"
    ];

    wantedBy = [
      "multi-user.target"
    ];

    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;

      User = "root";
      Group = "root";

      NoNewPrivileges = true;
      PrivateTmp = true;
      PrivateDevices = true;

      ProtectSystem = "strict";
      ProtectHome = true;

      ProtectKernelTunables = true;
      ProtectKernelModules = true;
      ProtectKernelLogs = true;
      ProtectControlGroups = true;

      RestrictSUIDSGID = true;
      RestrictRealtime = true;
      RestrictNamespaces = true;

      LockPersonality = true;

      CapabilityBoundingSet = [
        "CAP_NET_ADMIN"
      ];

      RestrictAddressFamilies = [
        "AF_NETLINK"
        "AF_UNIX"
      ];
    };

    script = ''
      set -euo pipefail

      nft=${pkgs.nftables}/bin/nft

      # Remove only our own table from a previous invocation. Never flush
      # or modify the machine-wide firewall ruleset.
      if "$nft" list table inet gjallar_ai_model >/dev/null 2>&1; then
        "$nft" delete table inet gjallar_ai_model
      fi

      "$nft" -f - <<'EOF'
      table inet gjallar_ai_model {
        chain output {
          type filter hook output priority -50; policy accept;

          ip daddr 127.0.0.1 tcp dport 11434 meta skuid ollama accept
          ip daddr 127.0.0.1 tcp dport 11434 meta skuid gjallar-ai-model accept

          ip daddr 127.0.0.1 tcp dport 11434 counter reject
        }
      }
      EOF

      # Fail closed if the expected reject rule is not actually present.
      "$nft" list table inet gjallar_ai_model \
        | ${pkgs.gnugrep}/bin/grep -F \
            'ip daddr 127.0.0.1 tcp dport 11434 counter packets 0 bytes 0 reject' \
            >/dev/null \
        || {
          echo "ERROR: Ollama firewall rule was not installed." >&2
          "$nft" delete table inet gjallar_ai_model 2>/dev/null || true
          exit 1
        }
    '';

    preStop = ''
      ${pkgs.nftables}/bin/nft \
        delete table inet gjallar_ai_model \
        2>/dev/null \
        || true
    '';
  };

  # Security dependency, not merely ordering:
  # a failed/missing firewall gate prevents Ollama from starting.
  systemd.services.ollama.requires = [
    "gjallar-ai-model-firewall.service"
  ];

  systemd.services.ollama.after = [
    "gjallar-ai-model-firewall.service"
  ];


  security.polkit.extraConfig = lib.mkAfter ''
    polkit.addRule(function(action, subject) {
      if (
        action.id !=
        "org.freedesktop.systemd1.manage-units"
      ) {
        return polkit.Result.NOT_HANDLED;
      }

      if (
        !subject.active ||
        !subject.local ||
        subject.user != "${settings.username}"
      ) {
        return polkit.Result.NOT_HANDLED;
      }

      var unit = action.lookup("unit");
      var verb = action.lookup("verb");

      if (
        typeof unit !== "string" ||
        !/^gjallar-ai-session@[^/]+\\.service$/.test(unit)
      ) {
        return polkit.Result.NOT_HANDLED;
      }

      if (
        verb == "start" ||
        verb == "stop"
      ) {
        return polkit.Result.YES;
      }

      return polkit.Result.NO;
    });
  '';

  systemd.services.gjallar-ai-model = {
    description =
      "GjallarOS inference-only model broker";

    after = [
      "ollama.service"
    ];

    requires = [
      "ollama.service"
    ];

    wantedBy = [
      "multi-user.target"
    ];

    serviceConfig = {
      ExecStart =
        "${pkgs.gjallarctl or (pkgs.callPackage ../../pkgs/gjallarctl { })}/bin/gjallarctl "
        + "ai model-serve "
        + "--socket /run/gjallar-ai-model/model.sock "
        + "--upstream http://127.0.0.1:11434 "
        + "--model ${assistantModel} "
        + "--user ${lib.escapeShellArg settings.username} "
        + "--cgroup-prefix /system.slice/gjallar-ai-session@ "
        + "--peer-exe ${pkgs.socat}/bin/socat";

      User = "gjallar-ai-model";
      Group = "gjallar-ai-model";

      RuntimeDirectory = "gjallar-ai-model";
      RuntimeDirectoryMode = "0750";

      NoNewPrivileges = true;
      PrivateTmp = true;
      PrivateDevices = true;

      ProtectSystem = "strict";
      ProtectHome = true;

      ProtectKernelTunables = true;
      ProtectKernelModules = true;
      ProtectKernelLogs = true;
      ProtectControlGroups = true;

      RestrictSUIDSGID = true;
      RestrictRealtime = true;
      RestrictNamespaces = true;

      CapabilityBoundingSet = "";
      LockPersonality = true;

      RestrictAddressFamilies = [
        "AF_UNIX"
        "AF_INET"
        "AF_INET6"
      ];
    };
  };

  systemd.services."gjallar-ai-session@" = {
    description =
      "GjallarOS controlled AI session for %i";

    after = [
      "gjallar-ai-model.service"
      "ollama-model-provision.service"
    ];

    requires = [
      "gjallar-ai-model.service"
      "ollama.service"
    ];

    serviceConfig = {
      Type = "simple";

      ExecStart =
        "${aiSessionRuntime} %i";

      User = "root";
      Group = "root";

      RuntimeDirectory =
        "gjallar-ai-session-%i";

      RuntimeDirectoryMode = "0755";

      UMask = "0077";

      PrivateNetwork = true;
      PrivateTmp = true;
      PrivateDevices = true;
      PrivateMounts = true;

      # Decode the systemd-escaped absolute instance path and expose only
      # that Git workspace inside the service namespace.
      BindPaths = [
        "%f:/workspace"
      ];

      # Hide all real home directories after systemd has established the
      # explicit workspace bind.
      TemporaryFileSystem = [
        "/home:mode=0755,nosuid,nodev"
      ];

      NoNewPrivileges = true;

      ProtectSystem = "strict";
      ProtectHome = false;

      ProtectKernelTunables = true;
      ProtectKernelModules = true;
      ProtectKernelLogs = true;
      ProtectControlGroups = true;

      RestrictSUIDSGID = true;
      RestrictRealtime = true;
      RestrictNamespaces = true;

      LockPersonality = true;

      CapabilityBoundingSet = [
        "CAP_SETUID"
        "CAP_SETGID"
      ];

      RestrictAddressFamilies = [
        "AF_UNIX"
        "AF_INET"
        "AF_INET6"
      ];

      TimeoutStopSec = "10s";
    };
  };

  systemd.services.ollama-model-provision = {
    description = "Provision the resolved GjallarOS Ollama model";
    after = [ "ollama.service" ];
    requires = [ "ollama.service" ];
    wantedBy = [ "multi-user.target" ];
    path = [ ollamaPackage pkgs.curl pkgs.coreutils pkgs.gnugrep ];
    environment = {
      HOME = "/var/lib/ollama";
      OLLAMA_HOST = "http://127.0.0.1:11434";
      OLLAMA_MODELS = "/var/lib/ollama/models";
    };
    serviceConfig = {
      Type = "oneshot";
      User = "ollama";
      Group = "ollama";
      StateDirectory = "gjallaros-agent";
      StateDirectoryMode = "0755";
      TimeoutStartSec = "30min";
      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = true;
      ReadWritePaths = [ "/var/lib/ollama" ];
    };
    script = ''
      set -euo pipefail
      ready=false
      for _ in $(seq 1 30); do
        if curl --silent --fail --max-time 2 http://127.0.0.1:11434/api/tags >/dev/null; then ready=true; break; fi
        sleep 1
      done
      if [ "$ready" != true ]; then echo "ERROR: Ollama did not become ready within 30 seconds." >&2; exit 1; fi
      ollama pull ${lib.escapeShellArg settings.aiModel}
      state="$STATE_DIRECTORY/definition.sha256"
      applied="$(cat "$state" 2>/dev/null || true)"
      present=false
      if ollama list | grep -q '^${assistantModel}[[:space:]]'; then present=true; fi
      if [ "$present" != true ] || [ "$applied" != ${lib.escapeShellArg modelDefinitionHash} ]; then
        ollama create ${assistantModel} --file ${modelDefinitionFile}
        printf '%s\n' ${lib.escapeShellArg modelDefinitionHash} > "$state"
      fi
    '';
  };

  systemd.services.gjallar-ai-research = {
    description = "GjallarOS read-only research broker";
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
      ExecStart = "${pkgs.gjallarctl or (pkgs.callPackage ../../pkgs/gjallarctl { })}/bin/gjallarctl ai research-serve";
      RuntimeDirectory = "gjallar-ai";
      RuntimeDirectoryMode = "0755";
      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = true;
      PrivateDevices = true;
      ProtectKernelTunables = true;
      ProtectKernelModules = true;
      ProtectControlGroups = true;
      RestrictSUIDSGID = true;
      RestrictAddressFamilies = [ "AF_UNIX" "AF_INET" "AF_INET6" ];
    };
  };

  users.groups.ollama = { };

  users.users.ollama = {
    isSystemUser = true;
    group = "ollama";
    home = "/var/lib/ollama";
    createHome = true;
  };

  services.ollama = rec {
    enable = true;

    user = "ollama";
    group = "ollama";

    # Keep the model API local; expose it deliberately through a reverse
    # proxy or VPN if remote access is ever required.
    openFirewall = false;
    host = "127.0.0.1";
    port = 11434;

    environmentVariables = ollamaEnvironment;

    package = ollamaPackage;
  };

  systemd.services.ollama.serviceConfig = {
    # Ollama must have a stable kernel UID because the local API firewall
    # identifies the originating socket by skuid.
    #
    # Current NixOS Ollama modules can still force DynamicUser=true even
    # when a static services.ollama.user is configured.
    DynamicUser = lib.mkForce false;

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

{
  description = "GjallarOS — a practical Nordic NixOS workstation";

  inputs = {
    superfile.url = "github:yorukot/superfile";

    nixpkgs.url = "github:nixos/nixpkgs/nixos-26.05";

    home-manager.url = "github:nix-community/home-manager/release-26.05";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";

    lanzaboote = {
      url = "github:nix-community/lanzaboote/v1.1.0";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    sops-nix.url = "github:Mic92/sops-nix";

    nur = {
      url = "github:nix-community/NUR";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    stylix = {
      url = "github:danth/stylix/release-26.05";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    caelestia.url = "github:caelestia-dots/shell";

    hyprland = {
      type = "git";
      url = "https://github.com/hyprwm/Hyprland";
      submodules = true;
    };

    aagl = {
      url = "github:ezKEa/aagl-gtk-on-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    nixvim.url = "github:nix-community/nixvim";

    hyprland-plugins = {
      url = "github:hyprwm/hyprland-plugins";
      inputs.hyprland.follows = "hyprland";
    };

    hypr-dynamic-cursors = {
      url = "github:VirtCode/hypr-dynamic-cursors";
      inputs.hyprland.follows = "hyprland";
    };

    plasma-manager = {
      url = "github:nix-community/plasma-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    zen-browser = {
      url = "github:0xc000022070/zen-browser-flake";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    winapps = {
      url = "github:winapps-org/winapps";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    opencode = {
      url = "github:anomalyco/opencode";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    noctalia-greeter = {
      url = "github:noctalia-dev/noctalia-greeter";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    noctalia = {
      url = "github:noctalia-dev/noctalia";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    late = {
      url = "github:mpiorowski/late-sh";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    # OS-neutral JODS agent protocol with the NixOS executor imported through
    # system/management/jods. Pinned source; never a developer-machine path.
    jods = {
      url = "git+https://github.com/bakanura/jods.git?rev=3673356b81109bfaf827eea0cc58888f1add8779&shallow=1";
      flake = false;
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      home-manager,
      ...
    }@inputs:
    let
      system = "x86_64-linux";
      lib = nixpkgs.lib;
      releasePolicy = builtins.fromJSON (builtins.readFile ./deployment/release-policy.json);
      sourceRevision =
        if self ? rev then
          "git:${self.rev}"
        else if self ? dirtyRev then
          "git:${self.dirtyRev}"
        else if self ? narHash then
          "nar:${self.narHash}"
        else
          throw "GjallarOS source provenance is unavailable";

      basePkgs = nixpkgs.legacyPackages.${system};

      settings = import (./. + "/settings.nix") {
        pkgs = basePkgs;
        inherit inputs;
      };

      superfileOverlay = final: prev: {
        superfile = inputs.superfile.packages.${system}.superfile.overrideAttrs (old: {
          nativeBuildInputs =
            (lib.filter (
              input:
              let
                name = lib.getName input;
              in
              name != "go" && !(lib.hasPrefix "go-" name)
            ) (old.nativeBuildInputs or [ ]))
            ++ [ final.go_1_26 ];

          env = (old.env or { }) // {
            GOTOOLCHAIN = "local";
          };

          meta = (old.meta or { }) // {
            mainProgram = "superfile";
          };
        });
      };

      pkgs = import nixpkgs {
        inherit system;
        overlays = [
          superfileOverlay
        ];
      };
    in
    {
      packages.${system} = rec {
        gjallarctl = pkgs.callPackage ./pkgs/gjallarctl { };
        "gjallar-installer" = gjallarctl.overrideAttrs (old: {
          meta = old.meta // {
            mainProgram = "gjallar-installer";
          };
        });
        "gjallar-recovery-iso" = self.nixosConfigurations.gjallar-recovery.config.system.build.isoImage;
        "gjallar-recovery-vm-iso" =
          self.nixosConfigurations.gjallar-recovery-vm.config.system.build.isoImage;
      };

      formatter = {

        x86_64-linux = nixpkgs.legacyPackages.x86_64-linux.nixfmt;

        aarch64-linux = nixpkgs.legacyPackages.aarch64-linux.nixfmt;

      };

      nixosConfigurations = {
        gjallar-recovery = nixpkgs.lib.nixosSystem {
          inherit system;
          modules = [ ./system/recovery/image.nix ];
          specialArgs = {
            releaseVersion = releasePolicy.release;
            repoSource = self.outPath;
            inherit settings sourceRevision;
          };
        };

        gjallar-recovery-vm = nixpkgs.lib.nixosSystem {
          inherit system;
          modules = [
            ./system/recovery/image.nix
            ./system/recovery/vm-image.nix
          ];
          specialArgs = {
            releaseVersion = releasePolicy.release;
            repoSource = self.outPath;
            inherit settings sourceRevision;
          };
        };

        ${settings.hostname} = nixpkgs.lib.nixosSystem {
          modules = [
            ./system/apps/brave-backend.nix
            ./system/security/secure-boot
            inputs.noctalia-greeter.nixosModules.default
            inputs.stylix.nixosModules.stylix
            inputs.sops-nix.nixosModules.sops
            inputs.home-manager.nixosModules.home-manager
            inputs.lanzaboote.nixosModules.lanzaboote

            {
              nixpkgs.overlays = [
                inputs.nur.overlays.default
                superfileOverlay
              ];

              home-manager.useGlobalPkgs = true;
              home-manager.useUserPackages = true;
              home-manager.backupFileExtension = "hm-bak";

              home-manager.extraSpecialArgs = {
                inherit inputs settings;
              };

              home-manager.sharedModules = [
                inputs.plasma-manager.homeModules.plasma-manager
                inputs.nixvim.homeModules.nixvim
                inputs.sops-nix.homeManagerModules.sops
                inputs.zen-browser.homeModules.twilight
                inputs.noctalia.homeModules.default
              ];

              home-manager.users = {
                ${settings.username} = import (./. + "/profiles/${settings.profile}/home.nix");
              }
              // nixpkgs.lib.optionalAttrs settings.workUserEnable {
                ${settings.workUsername} = import ./profiles/work-user/home.nix;
              };

              systemd.services.display-manager.after = [
                "home-manager-${settings.username}.service"
              ];

              systemd.services.display-manager.wants = [
                "home-manager-${settings.username}.service"
              ];

              systemd.services.display-manager.requires = [
                "home-manager-${settings.username}.service"
              ];
            }

            (./. + "/profiles/${settings.profile}/configuration.nix")
          ]
          ++ nixpkgs.lib.optionals (settings.endpointManagedDevice or false) [
            ./system/management/jods
            "${inputs.jods}/nix/modules/jods-mdm-agent.nix"
          ];

          specialArgs = {
            inherit inputs settings;
          };
        };
      };

      homeConfigurations = {
        ${settings.username} = home-manager.lib.homeManagerConfiguration {
          pkgs = nixpkgs.legacyPackages.${settings.system};

          modules = [
            (./. + "/profiles/${settings.profile}/home.nix")
            inputs.plasma-manager.homeModules.plasma-manager
            inputs.stylix.homeModules.stylix
            inputs.nixvim.homeModules.nixvim
            inputs.sops-nix.homeManagerModules.sops
            inputs.zen-browser.homeModules.twilight
            inputs.noctalia.homeModules.default
          ];

          extraSpecialArgs = {
            inherit inputs settings;
          };
        };
      }
      // nixpkgs.lib.optionalAttrs settings.workUserEnable {
        ${settings.workUsername} = home-manager.lib.homeManagerConfiguration {
          pkgs = nixpkgs.legacyPackages.${settings.system};

          modules = [
            ./profiles/work-user/home.nix
            inputs.plasma-manager.homeModules.plasma-manager
            inputs.stylix.homeModules.stylix
            inputs.nixvim.homeModules.nixvim
            inputs.sops-nix.homeModules.sops
            inputs.zen-browser.homeModules.twilight
            inputs.noctalia.homeModules.default
          ];

          extraSpecialArgs = {
            inherit inputs settings;
          };
        };
      };
    };
}

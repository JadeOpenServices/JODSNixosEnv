{
  description = "GjallarOS — a practical Nordic NixOS workstation";

  inputs = {
    yazi-disk-space = {
      url = "github:shafayetejaman/sduf.yazi";
      flake = false;
    };

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

    noctalia-greeter = {
      url = "github:noctalia-dev/noctalia-greeter";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    noctalia = {
      url = "github:noctalia-dev/noctalia";
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

      overlays = [
        inputs.nur.overlays.default
      ]
      ++ import ./pkgs/lib/overlays.nix;

      mkPkgs =
        targetSystem:
        import nixpkgs {
          system = targetSystem;
          inherit overlays;
          config.allowUnfree = true;
        };

      basePkgs = mkPkgs system;

      settings = import (./. + "/generated/state.nix") {
        pkgs = basePkgs;
        inherit inputs;
      };

      installState = import (./. + "/generated/install-state.nix");

      pkgs = basePkgs;
    in
    {
      nixosModules = import ./oddc/nixos/registry.nix;

      packages.${system} = rec {
        gjallarctl = pkgs.callPackage ./pkgs/gjallarctl { };
        "gjallar-installer" = gjallarctl.overrideAttrs (old: {
          meta = old.meta // {
            mainProgram = "gjallar-installer";
          };
        });
        "gjallar-recovery-iso" = self.nixosConfigurations.gjallar-recovery.config.system.build.isoImage;
        "gjallar-installer-lab-iso" =
          self.nixosConfigurations.gjallar-installer-lab.config.system.build.isoImage;
      };

      checks.${system} = {
        m620-legacy-nvidia-policy = import ./tests/nix/m620-policy.nix {
          inherit nixpkgs system;
          graphicsModule = ./system/hardware/graphics;
        };

        framework-battery-policy = import ./tests/nix/battery-policy.nix {
          inherit pkgs;
        };
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

        gjallar-installer-lab = nixpkgs.lib.nixosSystem {
          inherit system;
          modules = [
            ./system/recovery/image.nix
            ./system/recovery/installer-lab.nix
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
            ./pkgs/monique/nix/nixos-module.nix
            inputs.lanzaboote.nixosModules.lanzaboote

            {
              nixpkgs.overlays = [
                inputs.nur.overlays.default
              ];

              home-manager.useGlobalPkgs = true;
              home-manager.useUserPackages = true;
              home-manager.backupFileExtension = "hm-bak";

              home-manager.extraSpecialArgs = {
                inherit inputs settings installState;
              };

              home-manager.sharedModules = [
                inputs.plasma-manager.homeModules.plasma-manager
                inputs.sops-nix.homeManagerModules.sops
                inputs.zen-browser.homeModules.twilight
                inputs.noctalia.homeModules.default
              ];

              home-manager.users = {
                ${settings.username} = import ./user/default.nix;
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

            ./system/default.nix
          ]
          ++ nixpkgs.lib.optionals (settings.endpointManagedDevice or false) [
            ./system/management/jods
            "${inputs.jods}/nix/modules/jods-mdm-agent.nix"
          ];

          specialArgs = {
            inherit
              inputs
              settings
              sourceRevision
              installState
              ;
          };
        };
      };

      homeConfigurations = {
        ${settings.username} = home-manager.lib.homeManagerConfiguration {
          pkgs = mkPkgs settings.system;

          modules = [
            ./user/default.nix
            inputs.plasma-manager.homeModules.plasma-manager
            inputs.stylix.homeModules.stylix
            inputs.sops-nix.homeManagerModules.sops
            inputs.zen-browser.homeModules.twilight
            inputs.noctalia.homeModules.default
          ];

          extraSpecialArgs = {
            inherit inputs settings installState;
          };
        };
      };
    };
}

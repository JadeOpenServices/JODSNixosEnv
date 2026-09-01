{
    description = "GjallarOS — a practical Nordic NixOS workstation";

    outputs = { self, nixpkgs, home-manager, ... } @ inputs: let
        settings = import (./. + "/settings.nix") {inherit pkgs inputs;};
        pkgs = import nixpkgs {system = settings.system;};
    in {
        # NixOS configuration entrypoint.
        # 'nixos-rebuild switch --flake .#hostname
        nixosConfigurations = {
            ${settings.hostname} = nixpkgs.lib.nixosSystem {
                modules = [
                    (./. + "/system/compat/regreet-2605.nix")
                    inputs.stylix.nixosModules.stylix
                    inputs.sops-nix.nixosModules.sops
                    inputs.home-manager.nixosModules.home-manager
                    {
                        nixpkgs.overlays = [ inputs.nur.overlays.default ];
                        # Activate the matching Home Manager profile as part
                        # of the system rebuild. Without this, greetd can
                        # start Hyprland before its user configuration exists.
                        home-manager.useGlobalPkgs = true;
                        home-manager.useUserPackages = true;
                        # Preserve pre-existing user files when activating HM
                        # (for example VSCodium and xdg user-dir files).
                        home-manager.backupFileExtension = "hm-bak";
                        home-manager.extraSpecialArgs = { inherit inputs settings; };
                        home-manager.sharedModules = [
                            inputs.plasma-manager.homeModules.plasma-manager
                            inputs.nixvim.homeModules.nixvim
                            inputs.sops-nix.homeManagerModules.sops
                            inputs.zen-browser.homeModules.twilight
                            inputs.noctalia.homeModules.default
                    inputs.noctalia-greeter.nixosModules.default
                        ];
                        home-manager.users = {
                            ${settings.username} = import
                                (./. + "/profiles/${settings.profile}/home.nix");
                        } // nixpkgs.lib.optionalAttrs settings.workUserEnable {
                            ${settings.workUsername} = import ./profiles/work-user/home.nix;
                        };
                        # Do not offer a login prompt until the user
                        # configuration (including Hyprland) is activated.
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
                    # inputs.chaotic.nixosModules.default
                    (./. + "/profiles" + ("/" + settings.profile) + "/configuration.nix")
                ];
                specialArgs = {
                    inherit inputs;
                    inherit settings;
                };
            };
        };

        # Standalone home-manager configuration entrypoint.
        # 'home-manager switch --flake .#username
        homeConfigurations = {
            ${settings.username} = home-manager.lib.homeManagerConfiguration {
                pkgs = nixpkgs.legacyPackages.${settings.system};
                modules = [
                    (./. + "/profiles" + ("/" + settings.profile) + "/home.nix")
                    inputs.plasma-manager.homeModules.plasma-manager
                    inputs.stylix.homeModules.stylix
                    inputs.nixvim.homeModules.nixvim
                    inputs.sops-nix.homeManagerModules.sops
                    inputs.zen-browser.homeModules.twilight
                    inputs.noctalia.homeModules.default
                    inputs.noctalia-greeter.nixosModules.default
                    # inputs.chaotic.homeModules.default
                ];
                extraSpecialArgs = {
                    inherit inputs;
                    inherit settings;
                };
            };
        } // nixpkgs.lib.optionalAttrs settings.workUserEnable {
            ${settings.workUsername} = home-manager.lib.homeManagerConfiguration {
                pkgs = nixpkgs.legacyPackages.${settings.system};
                modules = [
                    (./. + "/profiles/work-user/home.nix")
                    inputs.plasma-manager.homeModules.plasma-manager
                    inputs.stylix.homeModules.stylix
                    inputs.nixvim.homeModules.nixvim
                    inputs.sops-nix.homeManagerModules.sops
                    inputs.zen-browser.homeModules.twilight
                    inputs.noctalia.homeModules.default
                    inputs.noctalia-greeter.nixosModules.default
                ];
                extraSpecialArgs = { inherit inputs settings; };
            };
        };
    };

    inputs = {
        # Keep these literals aligned with deployment/release-policy.json.
        # Flake input URLs must be static strings and cannot be computed from JSON.
        nixpkgs.url = "github:nixos/nixpkgs/nixos-26.05";
        home-manager.url = "github:nix-community/home-manager/release-26.05";
        home-manager.inputs.nixpkgs.follows = "nixpkgs";
        sops-nix.url = "github:Mic92/sops-nix";
        nur = {
            url = "github:nix-community/NUR";
            inputs.nixpkgs.follows = "nixpkgs";
        };
        # chaotic.url = "github:chaotic-cx/nyx/nyxpkgs-unstable";
        stylix = {
            url = "github:danth/stylix/release-26.05";
            inputs.nixpkgs.follows = "nixpkgs";
        };
        ags.url = "git+https://github.com/Aylur/ags?rev=60180a184cfb32b61a1d871c058b31a3b9b0743d";
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
        nixvim = {
            # url = "git+https://github.com/nix-community/nixvim?rev=f4b9a7122425c56d65466fcafb99053730b2646a";
            url = "github:nix-community/nixvim";
            # inputs.nixpkgs.follows = "nixpkgs";
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
    };
}

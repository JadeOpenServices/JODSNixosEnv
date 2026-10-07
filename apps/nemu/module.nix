# NixOS module from nemu (https://github.com/nemuTUI/nemu),
# pkg/linux/nixos/module.nix at 272a1aa04418d35b6933c8962779b6e5acb4c772,
# unchanged. It keeps its upstream licence:
#
# Copyright (c) 2014-2022, Pavel Balaev
# All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions are met:
#
# 1. Redistributions of source code must retain the above copyright notice,
#    this list of conditions and the following disclaimer.
# 2. Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
# AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
# ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
# LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
# CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
# SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
# INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
# CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
# ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
# POSSIBILITY OF SUCH DAMAGE.

{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.nemu;

in {
  options.programs.nemu = {
    enable = mkOption {
      type = types.bool;
      default = false;
      description = ''
        Whether to add nemu to the global environment and configure a
        setcap wrapper for it.
      '';
    };

    package = mkOption {
      type = types.package;
      default = pkgs.nemu;
      description = ''
        The package to use.
      '';
    };

    vhostNetGroup = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "vhost";
      description = ''
        Group for /dev/vhost-net. Will be created and udev rule will be added.
      '';
    };

    macvtapGroup = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "vhost";
      description = ''
        Group for /dev/tapN. Will be created and udev rule will be added.
      '';
    };

    usbGroup = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "usb";
      description = ''
        Group for USB devices. Will be created and udev rule will be added.
      '';
    };

    users = mkOption {
      default = {};
      example = {
        username_1 = {
          autoAddVeth = true;
          autoStartVMs = [ "vm1" "vm2" "vm3" ];
        };
        username_2 = {
          autoAddVeth = true;
          autoStartVMs = [ "vm" ];
        };
      };
      description = ''
        Users which will be able to run nemu.
        They will be added to vhostNetGroup, macvtapGroup,
        usbGroup and kvm group.
      '';
      type = types.attrsOf (types.submodule {
        options = {
          autoAddVeth = mkOption {
            type = types.bool;
            default = false;
            description = ''
              Whether to automatically create veth interfaces for user
              during system startup.
            '';
          };

          autoStartVMs = mkOption {
            type = types.listOf types.str;
            default = [];
            example = [ "vm1" "vm2" "vm3" ];
            description = ''
              List of VMs which will be started automatically for user
              during system startup.
            '';
          };
        };
      });
    };
  };

  config = mkIf cfg.enable {
    environment.systemPackages = with pkgs; [ cfg.package ];

    # Set capabilities
    security.wrappers.nemu = {
      source = "${cfg.package}/bin/nemu";
      capabilities = "cap_net_admin+ep";
    };

    # Add vhost-net, macvtap and usb groups
    users.groups =
      optionalAttrs (isString cfg.vhostNetGroup) {
        "${cfg.vhostNetGroup}" = { };
      } // optionalAttrs (isString cfg.macvtapGroup) {
        "${cfg.macvtapGroup}" = { };
      } // optionalAttrs (isString cfg.usbGroup) {
        "${cfg.usbGroup}" = { };
      };

    # Add nemu users to kvm, vhost-net, macvtap and usb groups
    users.users = mapAttrs' (user: _:
      nameValuePair "${user}" {
        extraGroups = [
          "kvm"
        ]
          ++ optional (isString cfg.vhostNetGroup) cfg.vhostNetGroup
          ++ optional (isString cfg.macvtapGroup) cfg.macvtapGroup
          ++ optional (isString cfg.usbGroup) cfg.usbGroup;
      }
    ) cfg.users;

    # Add udev rules for vhost-net, macvtap and usb groups
    services.udev.extraRules =
      optionalString (isString cfg.vhostNetGroup) ''
        KERNEL=="vhost-net", MODE="0660", GROUP="${cfg.vhostNetGroup}"
      '' + optionalString (isString cfg.macvtapGroup) ''
        SUBSYSTEM=="macvtap", MODE="0660", GROUP="${cfg.macvtapGroup}"
      '' + optionalString (isString cfg.usbGroup) ''
        SUBSYSTEM=="usb", MODE="0664", GROUP="${cfg.usbGroup}"
      '';

    # Add systemd nemu.target
    systemd.targets.nemu = {
      description = "nemu autostart target";
      wantedBy = [ "multi-user.target" ];
    };

    # Add systemd nemu-veth.target
    systemd.targets.nemu-veth = {
      description = "nemu veth creation target";
      before = [ "network-pre.target" ];
      wantedBy = [ "nemu.target" ];
    };

    # Add systemd nemu-vm.target
    systemd.targets.nemu-vm = {
      description = "nemu autostart target";
      wantedBy = [ "nemu.target" ];
    };

    # Add nemu-veth and nemu-vm systemd services for users
    systemd.services = mapAttrs' (user: _:
      nameValuePair "nemu-veth-${user}" {
        description = "Creates veth interfaces for nemu VMs for ${user}";
        serviceConfig = {
          Type = "oneshot";
          User = "${user}";
          ExecStart = "${config.security.wrapperDir}/nemu -c";
        };
        wantedBy = [ "nemu-veth.target" ];
      }
    ) (filterAttrs (_: userOpts: userOpts.autoAddVeth == true) cfg.users)
    // mapAttrs' (user: userOpts:
      nameValuePair "nemu-vm-${user}" {
        description = "Start nemu VMs for user ${user}";
        serviceConfig = {
          Type = "oneshot";
          User = "${user}";
          WorkingDirectory = "/home/${user}";
          RemainAfterExit = "yes";
          ExecStart = "${config.security.wrapperDir}/nemu --start "
            + concatStringsSep "," userOpts.autoStartVMs;
          ExecStop = "${config.security.wrapperDir}/nemu --poweroff "
            + concatStringsSep "," userOpts.autoStartVMs;
        };
        after = [
          "network-online.target"
          "nemu-veth-${user}.service"
        ];
        wantedBy = [ "nemu-vm.target" ];
      }
    ) (filterAttrs (_: userOpts: userOpts.autoStartVMs != []) cfg.users);
  };
}

{
  description = "Nix shell for Tarantool";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (import ../../pkgs/overlays/default.nix)
          ];
        };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            gdb
            cpulimit
            netcat-openbsd

            lua51Packages.lua
            lua51Packages.luacheck
            lua51Packages.luacov
            lua51Packages.luacheck
            lua51Packages.penlight
            unzip
            pkg-config
            zip

            cpulimit
            yq
            jq

            curl
            bc
            libyaml
            awscli

            parallel

            git
            gcc
            c-ares
            gnumake
            cmake
            nghttp2
            autoconf
            automake
            libtool
            readline
            ncurses
            openssl
            icu
            zlib
            python310
            python310Packages.pip # pyyaml, gevent, six
            lz4
            etcd

            go
            mage
            unzip

            protobuf
            protobufc

            nodejs

            libnl
            libpcap
            _msgpuck
          ];
          shellHook = ''
            export LD_LIBRARY_PATH=${pkgs.stdenv.cc.cc.lib}/lib
            export MSGPUCK_INCLUDE_DIR=${pkgs._msgpuck}/include
            export MSGPUCK_LIBRARY=${pkgs._msgpuck}/lib/libmsgpuck.a
            export LUA_INCDIR=${pkgs.lua51Packages.lua}/include
            export TARANTOOL_DIR=$HOME/Programming/tnt/tarantool/install/var/empty/local
            export TARANTOOL_INCDIR=$TARANTOOL_DIR/include
            export TT_CLI_TARANTOOL_PREFIX=$TARANTOOL_DIR
            export PATH=$TARANTOOL_DIR/bin:$PATH
            export PATH=$HOME/Programming/tnt/tarantool/test-run:$PATH
            export PATH=$HOME/Programming/tnt/tt:$PATH
            export PATH=$HOME/Programming/tnt/tt-ee:$PATH
            export PATH=$HOME/Programming/tnt/checkpatch:$PATH
            export PATH=$HOME/Programming/tnt/cartridge-cli:$PATH
            export PATH=$HOME/go/bin:$PATH
            export CC=${pkgs.gcc}/bin/gcc
            export CXX=${pkgs.gcc}/bin/c++
            source $HOME/Programming/tnt/.venv/bin/activate
            eval `ssh-agent -s`
            export ETCD_PATH=${pkgs.etcd}/bin
          '';

          hardeningDisable = [ "fortify" ];
        };
      }
    );
}

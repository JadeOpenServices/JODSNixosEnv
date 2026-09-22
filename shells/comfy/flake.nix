{
  description = "Nix shell for ComfyUI";

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
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            (pkgs.python311.withPackages (python-pkgs: [
              python-pkgs.pip
              python-pkgs.virtualenv
            ]))
          ];

          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath (
            with pkgs;
            [
              stdenv.cc.cc.lib
              zlib
              zstd
              glib
              libGL

              xorg.libxcb
              xorg.libX11
              xorg.libXext
              xorg.libXrender
            ]
          );

          shellHook = ''
            cd $HOME/Programming/ComfyUI
            source $HOME/Programming/ComfyUI/.venv/bin/activate
          '';

        };
      }
    );
}

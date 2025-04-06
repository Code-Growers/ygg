{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-parts.url = "github:hercules-ci/flake-parts";
    make-shell.url = "github:nicknovitski/make-shell";
  };

  outputs = inputs@{ nixpkgs, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.make-shell.flakeModules.default
      ];
      systems = [
        "x86_64-linux"
      ];

      perSystem = { config, self', inputs', pkgs, system, ... }:
        {
          make-shells.default = {
            packages = [
              pkgs.gopls
              pkgs.go
              pkgs.gotests
              pkgs.golangci-lint
            ];
          };
        };
    };
}


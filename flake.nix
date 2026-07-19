{
  description = "Alexandria — hermetic development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gotools
            golangci-lint
            buf
            jq
            just
          ];

          shellHook = ''
            export GOTOOLCHAIN=local
            echo "alexandria dev shell — go $(go version | cut -d' ' -f3), $(buf --version | head -1), golangci-lint $(golangci-lint version --short 2>/dev/null || true)"
          '';
        };
      });
    };
}

{
  description = "Developer shell flake for go 1.23 (supported for two releases)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11"; 
  };

  outputs = { self, nixpkgs }: {
    devShells.x86_64-linux.default = let 
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in 
      pkgs.mkShell {
        packages = [ 
          pkgs.go_1_26 
          pkgs.compile-daemon
          pkgs.jq
          pkgs.python3
          pkgs.lsof
          pkgs.codex
        ];
      };
  };
}


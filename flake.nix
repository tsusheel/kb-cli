{
  description = "Knowledge Base CLI tool for managing notes and tasks";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      pkgsFor = system: nixpkgs.legacyPackages.${system};
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.buildGoModule rec {
            pname = "kb";
            version = "0.1.12";

            src = ./.;

            # Exact Go vendor modules SRI hash
            vendorHash = "sha256-e8m/TiCWWh1T/EHhOdbcUW1XNRqVrFiVqOrmNVEemD4=";

            subPackages = [ "." ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
            ];

            postInstall = ''
              if [ -f $out/bin/kb-cli ] && [ ! -f $out/bin/kb ]; then
                mv $out/bin/kb-cli $out/bin/kb
              fi
              ln -s kb $out/bin/kb-cli
            '';

            meta = with pkgs.lib; {
              description = "Knowledge Base CLI tool for managing notes and tasks";
              homepage = "https://github.com/tsusheel/kb-cli";
              license = licenses.mit;
              mainProgram = "kb";
            };
          };
        });

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/kb";
        };
      });

      devShells = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
            ];
          };
        });
    };
}

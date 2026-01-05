{
  description = "LabWC theme changer (Bubble Tea TUI)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;

        runtimeBins = [
          pkgs.labwc
          pkgs.swww
          pkgs.kitty
          pkgs.glib
          pkgs.dconf
          pkgs.fuzzel
        ];
      in
      {
        packages.default = pkgs.buildGoModule rec {
          pname = "labwcchanger-tui";
          version = "0.1.0";
          src = self;

          # If this ever mismatches, set to lib.fakeHash, build once, paste the "got:" value.
          vendorHash = "sha256-5gvLdlLZq1H28gMvuQT/dp8ujt0rwtrz7xXLOvFbk0o=";

          # Be explicit; avoids surprises if your module layout changes.
          subPackages = [ "." ];

          buildInputs = [ pkgs.glib pkgs.gsettings-desktop-schemas pkgs.dconf ];
          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram "$out/bin/${pname}" \
              --prefix PATH : "${lib.makeBinPath runtimeBins}" \
              --set-default GSETTINGS_SCHEMA_DIR "${pkgs.gsettings-desktop-schemas}/share/gsettings-schemas/${pkgs.gsettings-desktop-schemas.name}/glib-2.0/schemas" \
              --set-default GSETTINGS_BACKEND "dconf"
          '';
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools git ];
        };
      }
    );
}

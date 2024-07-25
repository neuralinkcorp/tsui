{
  description = "A simple Go package";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";

  outputs = { self, nixpkgs }:
    let
      version = "0.0.1";

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          tsui = pkgs.buildGoModule {
            pname = "tsui";
            inherit version;
            # In 'nix develop', we don't need a copy of the source tree in the Nix store.
            src = ./.;

            # Inject the version info in the binary.
            ldflags = [
              "-X main.Version=${version}"
            ];

            # This hash locks the dependencies of this package. It is
            # necessary because of how Go requires network access to resolve
            # VCS.  See https://www.tweag.io/blog/2021-03-04-gomod2nix/ for
            # details. Normally one can build with a fake hash and rely on native Go
            # mechanisms to tell you what the hash should be or determine what
            # it should be "out-of-band" with other tooling (eg. gomod2nix).
            # Remember to bump this hash when your dependencies change.
            vendorHash = "sha256-758UX2EV1FKRlcTI8At16Wc6f2eUF2jSAaiMDFLax5I=";
          };
        });

      # Add dependencies that are only needed for development
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools go-tools ];
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.tsui);
    };
}

{
  description = "A simple Go package";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";

  outputs = { self, nixpkgs }:
    let
      version = "0.2.1";

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

      dependenciesFor = pkgs : with pkgs; []
        ++ (lib.optionals stdenv.isLinux [
          # For Linux clipboard support.
          xorg.libX11.dev
        ])
        ++ (lib.optionals stdenv.isDarwin [
          # For macOS clipboard support.
          darwin.apple_sdk.frameworks.Cocoa
        ]);
    in
    {
      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
          pname = "tsui";

          linuxInterpreters = {
            x86_64 = "/lib64/ld-linux-x86-64.so.2"; 
            aarch64 = "/lib/ld-linux-aarch64.so.1";
          };
          linuxInterpreter = linuxInterpreters.${pkgs.stdenv.hostPlatform.parsed.cpu.name};
        in
        rec {
          tsui = pkgs.buildGoModule {
            inherit pname;
            inherit version;
            src = ./.;

            # Possibly works around sporadic "signal: illegal instruction" error when
            # cross-compiling with macOS Rosetta.
            preBuild = if pkgs.stdenv.isLinux && pkgs.stdenv.isx86_64 then ''
              export GODEBUG=asyncpreemptoff=1
            '' else null;

            # Inject the version info in the binary.
            ldflags = [
              "-X main.Version=${version}"
            ];

            # This hash locks the dependencies of this package. It is
            # necessary because of how Go requires network access to resolve
            # VCS. See https://www.tweag.io/blog/2021-03-04-gomod2nix/ for
            # details. Normally one can build with a fake hash and rely on native Go
            # mechanisms to tell you what the hash should be or determine what
            # it should be "out-of-band" with other tooling (eg. gomod2nix).
            # Remember to bump this hash when your dependencies change.
            vendorHash = "sha256-FIbkPE5KQ4w7Tc7kISQ7ZYFZAoMNGiVlFWzt8BPCf+A=";

            buildInputs = dependenciesFor pkgs;
          };

          # This is an attempt at building a package independent from nix.
          # In order to do so, it changes the library loader for the one
          # usually found in `/lib/ld-linux....so`
          # Note that this does not change binary rpath, so libraries may still
          # be searched in `/nix/store`, but depending on the new ld-linux
          # used, it may also fallsback onto more "traditional" (e.g.
          # `/usr/lib64`) directories.
          # Note that this breaks the run on nixos-system, because
          # `/lib/ld-linux...` is not a real library loader.
          tsui_no_nix_ld = tsui.overrideAttrs (oldAttrs:
          {
            # Un-Nix the build so it can dlopen() X11 outside of Nix environments.
            preFixup = if pkgs.stdenv.isLinux then ''
              patchelf --remove-rpath --set-interpreter ${linuxInterpreter} $out/bin/${pname}
            '' else null;
          });
        });

      # Add dependencies that are only needed for development
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools go-tools ];
            nativeBuildInputs = dependenciesFor pkgs;
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.tsui);

      # nix bundle .# creates a file `tsui` in current directory which is a
      # self auto-extractable archive which should be self contained and hence
      # easy to deploy.
      bundles = forAllSystems (system: self.packages.${system}.tsui);
    };
}

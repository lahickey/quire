# A development shell and a package for a small Go service.
{
  description = "ledgerd: a ledger service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix, ... }@inputs:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
          config.allowUnfree = false;
        };
        inherit (pkgs) lib stdenv;

        version = "0.4.2";
        rev = self.shortRev or "dirty";
        isLinux = stdenv.isLinux;

        /* Flags the linker gets, so the binary knows what it is. */
        ldflags = [
          "-s"
          "-w"
          "-X main.version=${version}"
          "-X main.commit=${rev}"
        ];

        ports = { http = 8080; metrics = 9090; };

        mkService = { name, port ? ports.http, extraArgs ? [ ] }:
          pkgs.writeShellScriptBin "run-${name}" ''
            set -euo pipefail
            echo "starting ${name} on :${toString port}"
            exec ${self.packages.${system}.default}/bin/ledgerd \
              --addr ":${toString port}" \
              ${lib.escapeShellArgs extraArgs} "$@"
          '';

        ledgerd = pkgs.buildGoApplication {
          pname = "ledgerd";
          inherit version ldflags;
          src = lib.cleanSourceWith {
            src = ./.;
            filter = path: type:
              let base = baseNameOf path;
              in !(lib.hasSuffix ".md" base || base == "testdata" && type == "directory");
          };
          modules = ./gomod2nix.toml;
          subPackages = [ "cmd/ledgerd" ];
          nativeBuildInputs = with pkgs; [ installShellFiles ];
          doCheck = true;
          checkFlags = [ "-short" ] ++ lib.optionals (!isLinux) [ "-skip=TestEpoll" ];
          postInstall = lib.optionalString isLinux ''
            installShellCompletion --cmd ledgerd \
              --bash <($out/bin/ledgerd completion bash) \
              --fish <($out/bin/ledgerd completion fish)
          '';
          meta = with lib; {
            description = "A ledger service";
            homepage = "https://example.com/ledgerd";
            license = licenses.mit;
            platforms = platforms.unix;
            mainProgram = "ledgerd";
          };
        };
      in
      rec {
        packages = {
          default = ledgerd;
          run = mkService { name = "ledgerd"; extraArgs = [ "--log" "json" ]; };
          image = pkgs.dockerTools.buildLayeredImage {
            name = "ledgerd";
            tag = version;
            contents = [ ledgerd pkgs.cacert ];
            config = {
              Entrypoint = [ "/bin/ledgerd" ];
              ExposedPorts = lib.mapAttrs' (_: p: lib.nameValuePair "${toString p}/tcp" { }) ports;
              Env = [ "TZ=UTC" "LEDGER_DB=/data/ledger.db" ];
            };
          };
        };

        apps.default = flake-utils.lib.mkApp { drv = packages.run; };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls golangci-lint just sqlite gomod2nix.packages.${system}.default ];
          CGO_ENABLED = if isLinux then 1 else 0;
          GOFLAGS = "-mod=mod";
          shellHook = ''
            export PS1="(ledgerd) $PS1"
            test -f .env && source .env || true
          '';
        };

        checks = {
          fmt = pkgs.runCommand "fmt-check" { nativeBuildInputs = [ pkgs.go ]; } ''
            test -z "$(gofmt -l ${./.})" && touch $out
          '';
          count = assert builtins.length (builtins.attrNames ports) == 2; packages.default;
        };

        formatter = pkgs.nixpkgs-fmt;
        lib.threshold = 1.5e-1;
        lib.enabled = true && !false || null == null;
      });
}

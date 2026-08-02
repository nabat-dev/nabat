{
  description = "Nabat - Go CLI framework development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, git-hooks }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # nixpkgs' default `go` may lag; pin the toolchain Nabat requires.
        go = pkgs.go_1_27;

        graphifyApp = (import ./nix/graphifyy.nix { inherit pkgs; }).graphifyApp;

        # Keep `go` outside `with pkgs` so the pin is not shadowed by pkgs.go.
        devTools = [
          go
        ]
        ++ (with pkgs; [
          gopls
          gotools
          # nixpkgs golangci-lint is still built with Go 1.26 and rejects go.mod
          # 1.27 until upstream ships Go 1.27 support:
          # https://github.com/golangci/golangci-lint/issues/6643
          golangci-lint
          markdownlint-cli
          delve
          gnumake
          git
          vhs
        ]);

        mkApp =
          { name, description, script }:
          {
            type = "app";
            program = toString (pkgs.writeShellScript name script);
            meta = {
              mainProgram = name;
              inherit description;
            };
          };

        pre-commit-check = git-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            # Use the pinned Go toolchain's gofmt; git-hooks' default wrapper
            # still ships Go 1.26 and rejects generic methods.
            gofmt = {
              enable = true;
              entry = "${go}/bin/gofmt -l -w";
              files = "\\.go$";
            };
            # Disabled while go.mod is 1.27: nixpkgs golangci-lint is built with
            # Go 1.26 and fails config load. Re-enable when
            # https://github.com/golangci/golangci-lint/issues/6643 lands.
            golangci-lint = {
              enable = false;
              extraPackages = [ go ];
            };
            markdownlint = {
              enable = true;
              settings.configuration =
                builtins.fromJSON (builtins.readFile ./.markdownlint.json);
            };
            go-mod-tidy = {
              enable = true;
              name = "go-mod-tidy";
              entry = "${go}/bin/go mod tidy";
              files = "(\\.go|go\\.mod|go\\.sum)$";
              pass_filenames = false;
            };
          };
        };
      in
      {
        checks = {
          pre-commit = pre-commit-check;
        };

        packages.graphify = graphifyApp;

        devShells.default = pkgs.mkShell {
          name = "nabat";
          packages = devTools ++ pre-commit-check.enabledPackages ++ [ graphifyApp ];

          env = {
            GO111MODULE = "on";
            CGO_ENABLED = "1";
          };

          shellHook = ''
            ${pre-commit-check.shellHook}
            export GOPATH="''${GOPATH:-$HOME/go}"
            export PATH="$GOPATH/bin:$PATH"
            echo "nabat dev shell — $(go version)"
          '';
        };

        apps = {
          graphify = mkApp {
            name = "graphify";
            description = "Run graphify from the pinned graphifyy PyPI package";
            script = ''
              exec ${graphifyApp}/bin/graphify "$@"
            '';
          };

          fmt = mkApp {
            name = "fmt";
            description = "Format all Go files with gofmt";
            script = ''
              exec ${go}/bin/gofmt -w .
            '';
          };

          fmt-check = mkApp {
            name = "fmt-check";
            description = "Fail if any Go file needs gofmt (lists paths)";
            script = ''
              out=$(${go}/bin/gofmt -l .)
              if [ -n "$out" ]; then
                echo "::error::Unformatted Go files:" >&2
                echo "$out" >&2
                exit 1
              fi
            '';
          };

          tidy = mkApp {
            name = "tidy";
            description = "Run go mod tidy for the module";
            script = ''
              exec ${go}/bin/go mod tidy
            '';
          };

          # golangci-lint apps stay wired for when Go 1.27 support ships:
          # https://github.com/golangci/golangci-lint/issues/6643
          lint = mkApp {
            name = "lint";
            description = "Run golangci-lint";
            script = ''
              exec ${pkgs.golangci-lint}/bin/golangci-lint run ./...
            '';
          };

          lint-gomod = mkApp {
            name = "lint-gomod";
            description = "Run golangci-lint gomoddirectives on go.mod (CI hygiene)";
            script = ''
              exec ${pkgs.golangci-lint}/bin/golangci-lint run -c .golangci-gomod.yaml ./...
            '';
          };

          lint-md = mkApp {
            name = "lint-md";
            description = "Lint Markdown files with markdownlint";
            script = ''
              exec ${pkgs.markdownlint-cli}/bin/markdownlint '**/*.md'
            '';
          };

          test = mkApp {
            name = "test";
            description = "Run unit tests with -shuffle=on";
            script = ''
              exec ${go}/bin/go test -shuffle=on ./...
            '';
          };

          demo = mkApp {
            name = "demo";
            description = "Render all docs/tapes/*.tape into docs/assets/";
            script = ''
              mkdir -p docs/assets bin
              for dir in examples/*/; do
                ${go}/bin/go build -o "bin/$(basename "$dir")" "./$dir"
              done
              export PATH="$PWD/bin:$PATH"
              for tape in docs/tapes/*.tape; do
                ${pkgs.vhs}/bin/vhs "$tape"
              done
            '';
          };

          publish-demo = mkApp {
            name = "publish-demo";
            description = "Sync docs/assets/ to R2 (requires R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_ENDPOINT_URL, R2_BUCKET, R2_DEST_PATH)";
            script = ''
              : "''${R2_ACCESS_KEY_ID:?R2_ACCESS_KEY_ID is required}"
              : "''${R2_SECRET_ACCESS_KEY:?R2_SECRET_ACCESS_KEY is required}"
              : "''${R2_ENDPOINT_URL:?R2_ENDPOINT_URL is required}"
              : "''${R2_BUCKET:?R2_BUCKET is required}"
              : "''${R2_DEST_PATH:?R2_DEST_PATH is required}"

              export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
              export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
              export AWS_DEFAULT_REGION="auto"

              exec ${pkgs.awscli2}/bin/aws s3 sync docs/assets/ \
                "s3://$R2_BUCKET/$R2_DEST_PATH" \
                --endpoint-url "$R2_ENDPOINT_URL"
            '';
          };

          test-race = mkApp {
            name = "test-race";
            description = "Run tests with race detector and write coverage.out";
            script = ''
              # Nix Go only (go#75031). Example mains under examples/ are not test packages; including
              # them in one -coverpkg=./... run triggers "no such tool covdata" in CI.
              export GOTOOLCHAIN=local
              go="${go}/bin/go"
              mapfile -t testpkgs < <("$go" list ./... | grep -vE '/examples(/|$)' || true)
              if [ ''${#testpkgs[@]} -eq 0 ]; then
                echo "go list: no packages after excluding examples/" >&2
                exit 1
              fi
              exec "$go" test -race -shuffle=on -covermode=atomic \
                -coverpkg=./... -coverprofile=coverage.out -timeout 10m "''${testpkgs[@]}"
            '';
          };
        };
      }
    );
}

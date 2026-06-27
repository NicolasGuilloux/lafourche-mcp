{ pkgs, lib, config, ... }:

{
  # Outils de dev pour le projet lafourche-mcp
  packages = [
    pkgs.git
    pkgs.gnumake
    pkgs.golangci-lint
    pkgs.gotools # goimports
    pkgs.delve
    pkgs.curl
    pkgs.jq
  ];

  languages.go = {
    enable = true;
    package = pkgs.go;
  };

  env = {
    # Valeurs par défaut surchargeables (cf. internal/lafourche/config.go)
    LAFOURCHE_SHOP_DOMAIN = lib.mkDefault "shop.lafourche.fr";
    LAFOURCHE_API_VERSION = lib.mkDefault "2024-10";
    CGO_ENABLED = "0";
  };

  # Raccourcis: `devenv shell` puis ces commandes, ou `devenv tasks run`
  # Le code Go vit dans src/ ; les scripts s'y exécutent.
  scripts = {
    build.exec = "cd \"$DEVENV_ROOT/src\" && go build -o ../bin/lafourche ./cmd/lafourche";
    run-cli.exec = "cd \"$DEVENV_ROOT/src\" && go run ./cmd/lafourche \"$@\"";
    serve-stdio.exec = "cd \"$DEVENV_ROOT/src\" && go run ./cmd/lafourche mcp";
    serve-http.exec = "cd \"$DEVENV_ROOT/src\" && go run ./cmd/lafourche mcp http :8080";
    test.exec = "cd \"$DEVENV_ROOT/src\" && go test ./...";
    lint.exec = "cd \"$DEVENV_ROOT/src\" && golangci-lint run ./...";
    fmt.exec = "cd \"$DEVENV_ROOT/src\" && gofmt -w . && goimports -w .";
  };

  enterShell = ''
    echo "🍴 lafourche-mcp — go $(go version | awk '{print $3}')"
    echo "   scripts: build | run-cli | serve-stdio | serve-http | test | lint | fmt"
  '';

  enterTest = ''
    cd "$DEVENV_ROOT/src"
    go build ./...
    go test ./...
  '';
}

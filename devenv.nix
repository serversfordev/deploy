{ pkgs, lib, config, inputs, ... }:

{
  packages = [
    pkgs.git
    pkgs.go-task
    pkgs.golangci-lint
    pkgs.goreleaser
    pkgs.svu
  ];

  languages.go.enable = true;
}

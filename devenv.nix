{ pkgs, lib, config, inputs, ... }:

{
  packages = [
    pkgs.git
    pkgs.go-task
    pkgs.golangci-lint
    pkgs.goreleaser
  ];

  languages.go.enable = true;
}

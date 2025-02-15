{ pkgs, lib, config, inputs, ... }:

{
  packages = [
    pkgs.git
    pkgs.go-task
    pkgs.golangci-lint
  ];

  languages.go.enable = true;
}

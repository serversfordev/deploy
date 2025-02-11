{ pkgs, lib, config, inputs, ... }:

{
  packages = [ pkgs.git ];

  languages.go.enable = true;
}

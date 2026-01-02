{ pkgs, ... }:
{
  packages = [ pkgs.git ];

  languages.go.enable = true;
  languages.go.package = pkgs.go_1_25;
}

{ pkgs ? import <nixpkgs> {} }:
with pkgs; buildGoModule rec {
  name = "goboilerplate";
  src = ./..;
  vendorSha256 = null;
}

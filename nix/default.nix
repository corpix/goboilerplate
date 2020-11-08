{ pkgs ? import ./nixpkgs.nix {} }:
with pkgs; buildGoModule rec {
  name = "goboilerplate";
  src = ./..;
  vendorSha256 = null;
}
